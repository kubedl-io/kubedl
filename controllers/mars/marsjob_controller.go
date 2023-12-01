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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kubedliov1beta1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/core"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/eventhandler"
	"github.com/alibaba/kubedl/pkg/metrics"
	utilruntime "github.com/alibaba/kubedl/pkg/util/runtime"
)

var log = logf.Log.WithName("mars-controller")

const (
	controllerName = "MarsController"
)

func NewReconciler(mgr ctrl.Manager, config options.JobControllerConfiguration) *MarsJobReconciler {
	r := &MarsJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(kubedliov1beta1.MarsJobKind, r.Client), mgr.GetScheme())
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	return r
}

var _ reconcile.Reconciler = &MarsJobReconciler{}
var _ v1.ControllerInterface = &MarsJobReconciler{}

type MarsJobReconciler struct {
	client.Client
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
	ctrl        job_controller.JobController
	coordinator core.Coordinator
	utilruntime.EmptyScaleImpl
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=training.kubedl.io,resources=marsjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubedl.io,resources=marsjobs/status,verbs=get;update;patch

func (r *MarsJobReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch latest mars job instance.
	sharedMarsJob := &kubedliov1beta1.MarsJob{}
	err := r.ctrl.APIReader.Get(context.Background(), req.NamespacedName, sharedMarsJob)
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

	result, err := r.ctrl.ReconcileJobs(marsJob, marsJob.Spec.MarsReplicaSpecs, marsJob.Status.JobStatus, &marsJob.Spec.RunPolicy, nil, nil)
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
	if err = c.Watch(&source.Kind{Type: &kubedliov1beta1.MarsJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: job_controller.OnOwnerCreateFunc(r.scheme, kubedliov1beta1.ExtractMetaFieldsFromObject, log, r.coordinator, r.ctrl.Metrics),
		UpdateFunc: job_controller.OnOwnerUpdateFunc(r.scheme, kubedliov1beta1.ExtractMetaFieldsFromObject, log, r.coordinator),
		DeleteFunc: job_controller.OnOwnerDeleteFunc(r.ctrl, kubedliov1beta1.ExtractMetaFieldsFromObject, log),
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
	return kubedliov1beta1.SchemeGroupVersion.WithKind(kubedliov1beta1.MarsJobKind)
}

func (r *MarsJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return kubedliov1beta1.SchemeGroupVersion
}

func (r *MarsJobReconciler) GetGroupNameLabelValue() string {
	return kubedliov1beta1.SchemeGroupVersion.Group
}

func (r *MarsJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
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
		if podTemplate.Spec.Containers[i].Name == kubedliov1beta1.MarsJobDefaultContainerName {
			if len(podTemplate.Spec.Containers[i].Env) == 0 {
				podTemplate.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
			}

			cpuLimit := resourceRequestOrLimits(podTemplate.Spec.Containers[i].Resources, corev1.ResourceCPU)
			memLimit := resourceRequestOrLimits(podTemplate.Spec.Containers[i].Resources, corev1.ResourceMemory)
			envs := []corev1.EnvVar{
				{Name: "MARS_CPU_TOTAL", Value: strconv.Itoa(cpuLimit)},
				{Name: "MARS_MEMORY_TOTAL", Value: strconv.Itoa(memLimit)},
				{Name: "MARS_CPU_USE_PROCESS_STAT", Value: "1"},
				{Name: "MARS_MEM_USE_CGROUP_STAT", Value: "1"},
				{Name: "MARS_BIND_PORT", Value: strconv.Itoa(kubedliov1beta1.MarsJobDefaultPort)},
				{Name: "MARS_K8S_GROUP_LABELS", Value: v1.JobNameLabel},
				{Name: "MARS_CONTAINER_IP", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}},
				{Name: "MARS_K8S_POD_NAME", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
				{Name: "MARS_K8S_POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
				},
			}

			if cfg != "" {
				envs = append(envs, corev1.EnvVar{Name: marsConfig, Value: cfg})
			}

			// Convert memory tuning settings to environment variables so that worker processes
			// can be perceived.
			if strings.EqualFold(rtype, string(kubedliov1beta1.MarsReplicaTypeWorker)) && marsJob.Spec.WorkerMemoryTuningPolicy != nil {
				memTuningPolicy := marsJob.Spec.WorkerMemoryTuningPolicy
				if memTuningPolicy.SpillDirs != nil {
					injectSpillDirsByGivenPaths(memTuningPolicy.SpillDirs, podTemplate, &podTemplate.Spec.Containers[i])
					envs = append(envs, corev1.EnvVar{
						Name:  "MARS_SPILL_DIRS",
						Value: strings.Join(memTuningPolicy.SpillDirs, ","),
					})
				}
				if memTuningPolicy.PlasmaStore != nil && *memTuningPolicy.PlasmaStore != "" {
					envs = append(envs, corev1.EnvVar{
						Name:  "MARS_PLASMA_DIRS",
						Value: *memTuningPolicy.PlasmaStore,
					})
				}
				if memTuningPolicy.LockFreeFileIO != nil {
					lockFree := 0
					if *memTuningPolicy.LockFreeFileIO {
						lockFree = 1
					}
					envs = append(envs, corev1.EnvVar{
						Name:  "MARS_LOCK_FREE_FILEIO",
						Value: strconv.Itoa(lockFree),
					})
				}
				cacheSize := computeCacheMemSize(memLimit, memTuningPolicy)
				if cacheSize > 0 {
					envs = append(envs, corev1.EnvVar{
						Name:  "MARS_CACHE_MEM_SIZE",
						Value: strconv.Itoa(cacheSize),
					})
					mountPath := kubedliov1beta1.MarsJobDefaultCacheMountPath
					if memTuningPolicy.PlasmaStore != nil {
						mountPath = *memTuningPolicy.PlasmaStore
					}
					mountSharedCacheToPath(int64(cacheSize), mountPath, podTemplate, &podTemplate.Spec.Containers[i])
				}
			}
			appendOrOverrideEnvs(&podTemplate.Spec.Containers[i], envs...)
			break
		}
	}
	return nil
}

func (r *MarsJobReconciler) GetDefaultContainerName() string {
	return kubedliov1beta1.MarsJobDefaultContainerName
}

func (r *MarsJobReconciler) GetDefaultContainerPortName() string {
	return kubedliov1beta1.MarsJobDefaultPortName
}

func (r *MarsJobReconciler) GetDefaultContainerPortNumber() int32 {
	return kubedliov1beta1.MarsJobDefaultPort
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
func appendOrOverrideEnvs(container *corev1.Container, envs ...corev1.EnvVar) {
	// Map env name to its index for fast indexing.
	envIndexes := make(map[string]int)
	for i := range container.Env {
		envIndexes[container.Env[i].Name] = i
	}
	for ei := range envs {
		if index, ok := envIndexes[envs[ei].Name]; ok {
			container.Env[index] = envs[ei]
		} else {
			container.Env = append(container.Env, envs[ei])
		}
	}
}

// resourceRequestOrLimits prefers to take required resource from limits and then requests.
func resourceRequestOrLimits(rr corev1.ResourceRequirements, rname corev1.ResourceName) int {
	quant, ok := rr.Requests[rname]
	if ok {
		return int(quant.Value())
	}
	quant, ok = rr.Limits[rname]
	if ok {
		return int(quant.Value())
	}
	return 0
}

func (r *MarsJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	return ""
}
