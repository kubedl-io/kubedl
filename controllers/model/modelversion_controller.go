/*
Copyright 2021 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/controllers/model/storage"

	"github.com/go-logr/logr"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewModelVersionController(mgr ctrl.Manager, _ options.JobControllerConfiguration) *ModelVersionReconciler {
	return &ModelVersionReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ModelVersion"),
		Scheme: mgr.GetScheme(),
	}
}

// ModelVersionReconciler reconciles a ModelVersion object
type ModelVersionReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=model.kubedl.io,resources=modelversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=model.kubedl.io,resources=modelversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=model.kubedl.io,resources=models,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=model.kubedl.io,resources=models/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// ModelVersionController creates a kaniko container that builds a container image including the model.
func (r *ModelVersionReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the modelVersion
	modelVersion := &modelv1alpha1.ModelVersion{}
	err := r.Get(context.Background(), req.NamespacedName, modelVersion)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("modelVersion doesn't exist", "name", req.String())
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "fail to get model version "+req.String())
		return reconcile.Result{}, err
	}

	// no need to build
	if modelVersion.Status.ImageBuildPhase == modelv1alpha1.ImageBuildSucceeded ||
		modelVersion.Status.ImageBuildPhase == modelv1alpha1.ImageBuildFailed {
		r.Log.Info(fmt.Sprintf("image build %s", modelVersion.Status.ImageBuildPhase), "modelVersion", modelVersion.Name)
		return reconcile.Result{}, nil
	}

	// Does the model exist, create it if not.
	model := &modelv1alpha1.Model{}
	err = r.Get(context.Background(), types.NamespacedName{Namespace: modelVersion.Namespace, Name: modelVersion.Spec.ModelName}, model)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create model " + modelVersion.Spec.ModelName)
			model := &modelv1alpha1.Model{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: modelVersion.Namespace,
					Name:      modelVersion.Spec.ModelName,
				},
				Spec:   modelv1alpha1.ModelSpec{},
				Status: modelv1alpha1.ModelStatus{},
			}
			model.Status.LatestVersion = &modelv1alpha1.VersionInfo{
				ModelVersion: modelVersion.Name,
			}
			addModelToModelVersionOwnerReference(modelVersion, model)
			err := r.Create(context.Background(), model)
			if err != nil {
				r.Log.Error(err, "failed to create model "+model.Name)
				return reconcile.Result{Requeue: true}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		addModelToModelVersionOwnerReference(modelVersion, model)
	}

	pv := &v1.PersistentVolume{}
	pvc := &v1.PersistentVolumeClaim{}

	if modelVersion.Spec.Storage == nil {
		r.Log.Error(err, "storage is undefined", "modelversion", modelVersion.Name)
		return reconcile.Result{}, nil
	} else {
		err = r.createPVAndPVCForModelVersion(modelVersion, pv, pvc)
		if err != nil {
			r.Log.Error(err, "failed to create pv/pvc for modelversion", "modelversion", modelVersion.Name)
			return reconcile.Result{Requeue: true}, err
		}
	}

	// check if the pvc is bound
	if pvc.Status.Phase != v1.ClaimBound {
		// wait for the pv and pvc to be bound
		r.Log.Info("waiting for pv and pvc to be bound", "modelversion", modelVersion.Name, "pv", pv.Name, "pvc", pvc.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
	}

	modelVersionStatus := modelVersion.Status.DeepCopy()

	// create kaniko pod to build container image, take the preceding 5 chars for image versionId
	versionId := string(modelVersion.UID[:5])
	if len(modelVersion.Spec.ImageTag) > 0 {
		versionId = modelVersion.Spec.ImageTag
	}
	imgBuildPodName := GetBuildImagePodName(model.Name, versionId)
	imgBuildPod := &v1.Pod{}
	err = r.Get(context.Background(), types.NamespacedName{Namespace: model.Namespace, Name: imgBuildPodName}, imgBuildPod)
	if err != nil {
		if errors.IsNotFound(err) {
			modelVersionStatus.Image = modelVersion.Spec.ImageRepo + ":" + versionId
			modelVersionStatus.ImageBuildPhase = modelv1alpha1.ImageBuilding
			modelVersionStatus.Message = "Image building started."

			err = r.createDockerfileIfNotExists(model, modelVersion)
			if err != nil {
				return ctrl.Result{}, err
			}
			imgBuildPod = createImgBuildPod(modelVersion, pvc, imgBuildPodName, modelVersionStatus.Image)
			err = r.Create(context.Background(), imgBuildPod)
			if err != nil {
				return ctrl.Result{}, err
			}
			if modelVersion.Labels == nil {
				modelVersion.Labels = make(map[string]string, 1)
			}
			modelVersion.Labels["model.kubedl.io/model-name"] = model.Name
			if modelVersion.Annotations == nil {
				modelVersion.Annotations = make(map[string]string, 1)
			}
			modelVersion.Annotations["model.kubedl.io/img-build-pod-name"] = imgBuildPodName
			modelVersionCopy := modelVersion.DeepCopy()
			modelVersionCopy.Status = *modelVersionStatus
			// update model version
			err = r.Update(context.Background(), modelVersionCopy)
			if err != nil {
				r.Log.Error(err, "failed to update modelVersion")
				return ctrl.Result{}, err
			}
			modelCopy := model.DeepCopy()

			// update parent model latest version info
			modelCopy.Status.LatestVersion = &modelv1alpha1.VersionInfo{
				ModelVersion: modelVersion.Name,
				ImageName:    modelVersionStatus.Image,
			}
			err = r.Status().Update(context.Background(), modelCopy)
			if err != nil {
				r.Log.Error(err, "failed to update model", "mode", model.Name)
				return ctrl.Result{}, err
			}

			r.Log.Info(fmt.Sprintf("model %s image build started", modelVersion.Name), "image-build-pod-name", imgBuildPodName)
			return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	currentTime := metav1.Now()
	if imgBuildPod.Status.Phase == v1.PodSucceeded {
		modelVersionStatus.ImageBuildPhase = modelv1alpha1.ImageBuildSucceeded
		modelVersionStatus.Message = "Image build succeeded."
		modelVersionStatus.FinishTime = &currentTime
		r.Log.Info(fmt.Sprintf("modelversion %s image build succeeded", modelVersion.Name))
	} else if imgBuildPod.Status.Phase == v1.PodFailed {
		modelVersionStatus.ImageBuildPhase = modelv1alpha1.ImageBuildFailed
		modelVersionStatus.FinishTime = &currentTime
		modelVersionStatus.Message = "Image build failed."
		r.Log.Info(fmt.Sprintf("modelversion %s image build failed", modelVersion.Name))
	} else {
		// image not ready
		r.Log.Info(fmt.Sprintf("modelversion %s image is building", modelVersion.Name), "image-build-pod-name", imgBuildPodName)
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	}
	// image ready
	versionCopy := modelVersion.DeepCopy()
	versionCopy.Status = *modelVersionStatus
	err = r.Update(context.Background(), versionCopy)
	if err != nil {
		r.Log.Error(err, "failed to update modelVersion", "modelVersion", modelVersion.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// create the dockerfile to be used by kaniko
func (r *ModelVersionReconciler) createDockerfileIfNotExists(model *modelv1alpha1.Model,
	modelVersion *modelv1alpha1.ModelVersion) error {
	dockerfile := &v1.ConfigMap{}
	err := r.Get(context.Background(), types.NamespacedName{Namespace: model.Namespace, Name: "dockerfile"}, dockerfile)
	if err != nil {
		if errors.IsNotFound(err) {
			imgBuildDockerfile := createImageBuildDockerfile(modelVersion.Namespace)
			err = r.Create(context.Background(), imgBuildDockerfile)
		} else {
			return err
		}
	}
	return err
}

// createPVAndPVCForModelVersion creates pv and its pvc to be bound for specific model version.
// LocalStorage is treated different from remote storage. For local storage, each node will be provisioned with its
// own pv and pvc, and append nodeName as the pv/pvc name, whereas remote storage only has a single pv and pvc.
func (r *ModelVersionReconciler) createPVAndPVCForModelVersion(modelVersion *modelv1alpha1.ModelVersion,
	pv *v1.PersistentVolume, pvc *v1.PersistentVolumeClaim) (error error) {

	pvName := ""
	// for localstorage , append the nodeName
	if modelVersion.Spec.Storage.LocalStorage != nil {
		if modelVersion.Spec.Storage.LocalStorage.NodeName != "" {
			// get nodeName from modelversion
			pvName = GetModelVersionPVNameByNode(modelVersion.Name, modelVersion.Spec.Storage.LocalStorage.NodeName)
		} else {
			return fmt.Errorf("modelVersion has no nodeName set for local storage, modelVersion name %s", modelVersion.Name)
		}
	} else {
		// for remote storage
		pvName = GetModelVersionPVName(modelVersion.Name)
	}
	err := r.Get(context.Background(), types.NamespacedName{Name: pvName}, pv)
	if err != nil {
		if errors.IsNotFound(err) {
			// create a new pv for this particular model version
			storageProvider := storage.GetStorageProvider(modelVersion.Spec.Storage)
			pv = storageProvider.CreatePersistentVolume(modelVersion.Spec.Storage, pvName)
			if pv.OwnerReferences == nil {
				pv.OwnerReferences = make([]metav1.OwnerReference, 0)
			}
			pv.OwnerReferences = append(pv.OwnerReferences, metav1.OwnerReference{
				APIVersion: modelVersion.APIVersion,
				Kind:       modelVersion.Kind,
				Name:       modelVersion.Name,
				UID:        modelVersion.UID,
			})
			err := r.Create(context.Background(), pv)
			if err != nil {
				log.Info("failed to create pv for model version", "pv", pv.Name, "modelversion", modelVersion.Name)
				return err
			}
			log.Info("created pv for model version", "pv", pv.Name, "model-version", modelVersion.Name)
		} else {
			return err
		}
	}

	// create the pvc to the pv
	pvcName := ""
	// if using localstorage
	if modelVersion.Spec.Storage.LocalStorage != nil {
		if modelVersion.Spec.Storage.LocalStorage.NodeName != "" {
			// for modelversion, append the nodeName
			pvcName = GetModelVersionPVClaimNameByNode(modelVersion.Name, modelVersion.Spec.Storage.LocalStorage.NodeName)
		} else {
			return fmt.Errorf("both model and modelVersion don't have nodeName set for local storage, modelVersion name %s", modelVersion.Name)
		}
	} else {
		// for remote storage
		pvcName = GetModelVersionPVClaimName(modelVersion.Name)
	}
	// Does the pvc for the version already exist
	err = r.Get(context.Background(), types.NamespacedName{Namespace: modelVersion.Namespace, Name: pvcName}, pvc)
	if err != nil {
		if errors.IsNotFound(err) {
			pvc = createPVC(pv, pvcName, modelVersion.Namespace)
			if pvc.OwnerReferences == nil {
				pvc.OwnerReferences = make([]metav1.OwnerReference, 0)
			}
			pvc.OwnerReferences = append(pvc.OwnerReferences, metav1.OwnerReference{
				APIVersion: modelVersion.APIVersion,
				Kind:       modelVersion.Kind,
				Name:       modelVersion.Name,
				UID:        modelVersion.UID,
			})

			err = r.Create(context.Background(), pvc)
			if err != nil {
				log.Info("failed to create pvc for model version", "pvc", pvc.Name, "modelversion", modelVersion.Name)
				return err
			}
			log.Info("created pvc for model version", "pvc", pvc.Name, "modelversion", modelVersion.Name)
		} else {
			log.Error(err, fmt.Sprintf("modelversion %s failed too get pvc", modelVersion.Name))
			return err
		}
	}
	return err
}

func createPVC(pv *v1.PersistentVolume, name, namespace string) *v1.PersistentVolumeClaim {
	className := ""
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName:  pv.Name,
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("50Mi"),
				},
			},
			StorageClassName: &className,
		},
	}
	return pvc
}

// add model to modelversion's ownerReferences
func addModelToModelVersionOwnerReference(modelVersion *modelv1alpha1.ModelVersion, model *modelv1alpha1.Model) {
	// add owner reference
	if modelVersion.OwnerReferences == nil {
		modelVersion.OwnerReferences = make([]metav1.OwnerReference, 0)
	}
	exists := false
	for _, ref := range modelVersion.OwnerReferences {
		if ref.Kind == "Model" && ref.UID == model.UID {
			// already exists
			exists = true
			break
		}
	}
	if !exists {
		modelVersion.OwnerReferences = append(modelVersion.OwnerReferences, metav1.OwnerReference{
			APIVersion: model.APIVersion,
			Kind:       model.Kind,
			Name:       model.Name,
			UID:        model.UID,
		})
	}
}

// createImgBuildPod creates a kaniko pod to build the image
// 1. mount docker file into /workspace/dockerfile
// 2. mount build source pvc into /workspace/build
// 3. mount dockerconfig as /kaniko/.docker/config.json
// The image build pod will create a container that includes artifacts under "/workspace/build", based on the dockerfile
func createImgBuildPod(model *modelv1alpha1.ModelVersion, pvc *v1.PersistentVolumeClaim, imgBuildPodName string, newImage string) *v1.Pod {
	podSpec := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imgBuildPodName,
			Namespace: model.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  imgBuildPodName,
					Image: options.CtrlConfig.ModelImageBuilder,
					Args: []string{
						"--skip-tls-verify=true",
						"--dockerfile=/workspace/dockerfile",
						"--context=dir:///workspace/",
						fmt.Sprintf("--destination=%s", newImage)},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}
	var volumeMounts = []v1.VolumeMount{
		{
			Name:      "kaniko-secret", // the docker secret for pushing images
			MountPath: "/kaniko/.docker",
		},
		{
			Name:      "build-source", // build-source references the pvc for the model
			MountPath: "/workspace/build",
		},
		{
			Name:      "dockerfile", // dockerfile is the default Dockerfile for building the image including the model
			MountPath: "/workspace/",
		},
	}
	var volumes = []v1.Volume{
		{
			Name: "kaniko-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "regcred",
					Items: []v1.KeyToPath{
						{
							Key:  ".dockerconfigjson",
							Path: "config.json",
						},
					},
				},
			},
		},
		{
			Name: "build-source",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		},
		{
			Name: "dockerfile",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "dockerfile",
					},
				},
			},
		},
	}

	podSpec.Spec.Containers[0].VolumeMounts = volumeMounts
	podSpec.Spec.Volumes = volumes
	podSpec.OwnerReferences = append(podSpec.OwnerReferences, metav1.OwnerReference{
		APIVersion: model.APIVersion,
		Kind:       model.Kind,
		Name:       model.Name,
		UID:        model.UID,
	})
	return podSpec
}

// createImageBuildDockerfile creates a configmap mounted with dockerfile commands
// to build a minimum image contains model artifacts.
func createImageBuildDockerfile(ns string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockerfile",
			Namespace: ns,
		},
		Data: map[string]string{
			"dockerfile": fmt.Sprintf(`FROM busybox
    COPY build/ %s`, modelv1alpha1.DefaultModelPathInImage),
		},
	}
}

func (r *ModelVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var predicates = predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			version := event.Object.(*modelv1alpha1.ModelVersion)
			return version.DeletionTimestamp == nil
		},

		// DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
		//	modelVersion := deleteEvent.Meta.(*modelv1alpha1.ModelVersion)
		//
		//	// delete image build pod
		//	pod := &v1.Pod{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Namespace: modelVersion.Namespace,
		//			Name: modelVersion.Annotations["kubedl.io/img-build-pod-name"]},
		//	}
		//	err := r.Delete(context.Background(), pod)
		//	if err != nil {
		//		log.Error(err, "failed to delete pod "+pod.Name)
		//	}
		//	return true
		// },
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&modelv1alpha1.ModelVersion{}, builder.WithPredicates(predicates)).
		Complete(r)
}
