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

	"github.com/alibaba/kubedl/controllers/model/storage"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var log = logf.Log.WithName("model-controller")

// +kubebuilder:rbac:groups=model.kubedl.io,resources=models,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=model.kubedl.io,resources=models/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// ModelController reconciles and creates the underlying PV and PVC for the model
func (r *ModelReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	//Get the model
	model := &modelv1alpha1.Model{}
	err := r.Get(context.Background(), req.NamespacedName, model)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("model doesn't exist", "name", req.String())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	//TODO move to validation hook
	if model.Spec.Storage.LocalStorage != nil && model.Spec.Storage.LocalStorage.NodeName == "" {
		return reconcile.Result{}, fmt.Errorf("local storage should have node name set")
	}

	pv := &v1.PersistentVolume{}
	// Does the pv for the model already exist
	dirty := false
	err = r.Get(context.Background(), types.NamespacedName{Name: GetModelPVName(model.Name)}, pv)
	if err != nil {
		if errors.IsNotFound(err) {
			// create the pv and pvc for the model, if it doesn't exist
			storageProvider := storage.GetStorageProvider(model.Spec.Storage)
			pv = storageProvider.CreatePersistentVolume(model.Spec.Storage, GetModelPVName(model.Name))
			if pv.OwnerReferences == nil {
				pv.OwnerReferences = make([]metav1.OwnerReference, 0)
			}
			pv.OwnerReferences = append(pv.OwnerReferences, metav1.OwnerReference{
				APIVersion: model.APIVersion,
				Kind:       model.Kind,
				Name:       model.Name,
				UID:        model.UID,
			})
			err := r.Create(context.Background(), pv)
			if err != nil {
				return ctrl.Result{}, err
			}
			metav1.SetMetaDataAnnotation(&model.ObjectMeta, "model.kubedl.io/pv-name", pv.Name)
			dirty = true
			log.Info("created pv for model", "pv", pv.Name, "model", model.Name)
		} else {
			return reconcile.Result{}, err
		}
	}
	// Does the pvc for the model already exist
	pvc := &v1.PersistentVolumeClaim{}
	err = r.Get(context.Background(), types.NamespacedName{Namespace: model.Namespace, Name: GetModelPVCName(model.Name)}, pvc)
	if err != nil {
		if errors.IsNotFound(err) {
			// create the pvc to the pv
			pvc = createPVC(pv, GetModelPVCName(model.Name), model.Namespace)
			if pvc.OwnerReferences == nil {
				pvc.OwnerReferences = make([]metav1.OwnerReference, 0)
			}
			pvc.OwnerReferences = append(pvc.OwnerReferences, metav1.OwnerReference{
				APIVersion: model.APIVersion,
				Kind:       model.Kind,
				Name:       model.Name,
				UID:        model.UID,
			})
			metav1.SetMetaDataAnnotation(&model.ObjectMeta, "model.kubedl.io/pvc-name", pvc.Name)
			dirty = true
			err = r.Create(context.Background(), pvc)
			if err != nil {
				return ctrl.Result{}, err
			}
			log.Info("created pvc for model", "pvc", pvc.Name, "model", model.Name)
		} else {
			return reconcile.Result{}, err
		}
	}

	if dirty {
		err = r.Update(context.Background(), model)
	}
	return ctrl.Result{}, err
}

func createPVC(pv *v1.PersistentVolume, name, namespace string) *v1.PersistentVolumeClaim {
	className := "model-local"
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

func (r *ModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var modelPredicates = predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			model := event.Meta.(*modelv1alpha1.Model)
			if model.DeletionTimestamp != nil {
				return false
			}

			return true
		},

		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			//model := deleteEvent.Meta.(*modelv1alpha1.Model)
			//// delete pv
			//pv := &v1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: Get}}
			//err := r.Delete(context.Background(), pv)
			//if err != nil {
			//	log.Error(err, "failed to delete pv "+pv.Name)
			//}
			//
			//// delete pvc
			//pvc := &v1.PersistentVolumeClaim{
			//	ObjectMeta: metav1.ObjectMeta{Name: "model-pvc-" + model.Name},
			//}
			//err = r.Delete(context.Background(), pvc)
			//if err != nil {
			//	log.Error(err, "failed to delete pvc "+pvc.Name)
			//}
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&modelv1alpha1.Model{}, builder.WithPredicates(modelPredicates)).
		Complete(r)
}
