/*
Copyright 2020 The Alibaba Authors.

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
	"strconv"
	"strings"

	kubedliov1beta1 "github.com/alibaba/kubedl/apis/mars/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("mars-controller")

const (
	controllerName = "MarsController"
)

func NewReconciler(mgr ctrl.Manager, config job_controller.JobControllerConfiguration) *MarsJobReconciler {
	r := &MarsJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(r, config, r.recorder, metrics.NewJobMetrics(kubedliov1beta1.Kind, r.Client))
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	return r
}

var _ reconcile.Reconciler = &MarsJobReconciler{}
var _ v1.ControllerInterface = &MarsJobReconciler{}

type MarsJobReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	ctrl     job_controller.JobController
}

// +kubebuilder:rbac:groups=kubedl.io,resources=marsjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubedl.io,resources=marsjobs/status,verbs=get;update;patch

func (r *MarsJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Fetch latest mars job instance.
	sharedMarsJob := &kubedliov1beta1.MarsJob{}
	err := r.Get(context.Background(), req.NamespacedName, sharedMarsJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to get job but it has been deleted", "key", req.String())
			r.ctrl.Metrics.DeletedInc()
			return reconcile.Result{}, nil
		}
		// Error reading the object, requeue the request.
		return reconcile.Result{}, err
	}

	marsJob := sharedMarsJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(marsJob, marsJob.Spec.MarsReplicaSpecs)
	// No need to do reconciling or job has been deleted.
	if !needSync || marsJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to reconcile or has been deleted",
			"sync", needSync, "deleted", marsJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}

	r.scheme.Default(marsJob)

	result, err := r.ctrl.ReconcileJobs(marsJob, marsJob.Spec.MarsReplicaSpecs, marsJob.Status, &marsJob.Spec.RunPolicy)
	if err != nil {
		log.Error(err, "mars job reconcile failed")
		return result, err
	}

	if err = r.reconcileIngressForWebservice(marsJob); err != nil {
		log.Error(err, "ingress for webservices reconcile failed")
		return result, err
	}
	return result, nil
}

func (r *MarsJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
		Reconciler:              r,
	})
	if err != nil {
		return err
	}

	// Watch owner resources with create event filter.
	if err = c.Watch(&source.Kind{Type: &kubedliov1beta1.MarsJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: onOwnerCreateFunc(r),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &kubedliov1beta1.MarsJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &kubedliov1beta1.MarsJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	})
}

func (r *MarsJobReconciler) ControllerName() string {
	return controllerName
}

func (r *MarsJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return kubedliov1beta1.GroupVersionKind
}

func (r *MarsJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return kubedliov1beta1.SchemeGroupVersion
}

func (r *MarsJobReconciler) GetGroupNameLabelValue() string {
	return kubedliov1beta1.GroupName
}

func (r *MarsJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	marsJob, ok := job.(*kubedliov1beta1.MarsJob)
	if !ok {
		return fmt.Errorf("%+v is not type of MarsJob", job)
	}

	// Generate MARS_CONFIG json.
	cfg, err := marsConfigInJson(marsJob, rtype, index)
	if err != nil {
		return err
	}

	// Inject MARS_CONFIG env variable to mars container in the pod.
	for i := range podTemplate.Spec.Containers {
		if podTemplate.Spec.Containers[i].Name == kubedliov1beta1.DefaultContainerName {
			if len(podTemplate.Spec.Containers[i].Env) == 0 {
				podTemplate.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
			}
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "MARS_CPU_TOTAL",
				Value: strconv.Itoa(int(podTemplate.Spec.Containers[i].Resources.Limits.Cpu().Value())),
			}, corev1.EnvVar{
				Name:  "MARS_MEMORY_TOTAL",
				Value: strconv.Itoa(int(podTemplate.Spec.Containers[i].Resources.Limits.Memory().Value())),
			}, corev1.EnvVar{
				Name:  "MARS_CPU_USE_PROCESS_STAT",
				Value: "1",
			}, corev1.EnvVar{
				Name:  "MARS_MEM_USE_CGROUP_STAT",
				Value: "1",
			}, corev1.EnvVar{
				Name: "MARS_CONTAINER_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			}, corev1.EnvVar{
				Name:  "MARS_BIND_PORT",
				Value: strconv.Itoa(kubedliov1beta1.DefaultPort),
			}, corev1.EnvVar{
				Name:  "MARS_K8S_GROUP_LABELS",
				Value: v1.JobNameLabel,
			}, corev1.EnvVar{
				Name: "MARS_K8S_POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			}, corev1.EnvVar{
				Name: "MARS_K8S_POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			})

			if cfg != "" {
				podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  marsConfig,
					Value: cfg,
				})
			}

			// Convert memory tuning settings to environment variables so that worker processes
			// can be perceived.
			if v1.ReplicaType(rtype) == kubedliov1beta1.MarsReplicaTypeWorker && marsJob.Spec.WorkerMemoryTuningPolicy != nil {
				memTuningPolicy := marsJob.Spec.WorkerMemoryTuningPolicy
				if memTuningPolicy.SpillDirs != nil {
					injectSpillDirsByGivenPaths(memTuningPolicy.SpillDirs, podTemplate, &podTemplate.Spec.Containers[i])
					podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
						Name:  "MARS_SPILL_DIRS",
						Value: strings.Join(memTuningPolicy.SpillDirs, ","),
					})
				}
				if memTuningPolicy.PlasmaStore != nil {
					podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
						Name:  "MARS_PLASMA_DIRS",
						Value: *memTuningPolicy.PlasmaStore,
					})
				}
				if memTuningPolicy.WorkerCacheSize != nil {
					podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
						Name:  "MARS_CACHE_MEM_SIZE",
						Value: strconv.Itoa(int(memTuningPolicy.WorkerCacheSize.Value())),
					})
				} else if memTuningPolicy.WorkerCachePercentage != nil {
					podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
						Name:  "MARS_CACHE_MEM_SIZE",
						Value: strconv.Itoa(int(memTuningPolicy.WorkerCacheSize.Value())),
					})
				}
				if memTuningPolicy.LockFreeFileIO != nil {
					podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
						Name:  "MARS_LOCK_FREE_FILEIO",
						Value: strconv.FormatBool(*memTuningPolicy.LockFreeFileIO),
					})
				}
			}
			break
		}
	}
	return nil
}

func (r *MarsJobReconciler) GetDefaultContainerName() string {
	return kubedliov1beta1.DefaultContainerName
}

func (r *MarsJobReconciler) GetDefaultContainerPortName() string {
	return kubedliov1beta1.DefaultPortName
}

func (r *MarsJobReconciler) GetDefaultContainerPortNumber() int32 {
	return kubedliov1beta1.DefaultPort
}

func (r *MarsJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		kubedliov1beta1.MarsReplicaTypeScheduler,
		kubedliov1beta1.MarsReplicaTypeWebService,
		kubedliov1beta1.MarsReplicaTypeWorker,
	}
}

func (r *MarsJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	return false
}

func injectSpillDirsByGivenPaths(paths []string, template *corev1.PodTemplateSpec, container *corev1.Container) {
	if len(paths) == 0 {
		return
	}

	var sizeLimit *resource.Quantity

	if !container.Resources.Requests.StorageEphemeral().IsZero() {
		q := resource.MustParse(strconv.Itoa(int(container.Resources.Requests.StorageEphemeral().Value()) / len(paths)))
		sizeLimit = &q
	}

	for idx, path := range paths {
		volumeName := "mars-empty-dir-" + strconv.Itoa(idx)
		volume := corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		if sizeLimit != nil {
			volume.VolumeSource.EmptyDir.SizeLimit = sizeLimit
		}
		template.Spec.Volumes = append(template.Spec.Volumes, volume)

		// Check volume has not mounted yet inside container.
		existed := false
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == volumeName {
				existed = true
				break
			}
		}
		if existed {
			continue
		}

		// Mount empty dir to target path.
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: path,
		})
	}
}
