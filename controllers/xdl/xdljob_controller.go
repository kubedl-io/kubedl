/*
Copyright 2019 The Alibaba Authors.

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

package xdljob

import (
	"context"
	"fmt"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
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

const (
	controllerName = "XDLController"
)

const (
	// xdlConfig is the environment variable name of XDL cluster spec.
	xdlConfig = "XDL_CONFIG"
	taskType  = "TASK_NAME"
	taskIndex = "TASK_INDEX"
	zkAddr    = "ZK_ADDR"
)

var log = logf.Log.WithName("xdl-controller")

// newReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager, config options.JobControllerConfiguration) *XDLJobReconciler {
	r := &XDLJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(training.XDLJobKind, r.Client), mgr.GetScheme())
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	return r
}

var _ reconcile.Reconciler = &XDLJobReconciler{}
var _ v1.ControllerInterface = &XDLJobReconciler{}

// XDLJobReconciler reconciles a XDLJob object
type XDLJobReconciler struct {
	client.Client
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
	ctrl        job_controller.JobController
	coordinator core.Coordinator
	utilruntime.EmptyScaleImpl
}

// Reconcile reads that state of the cluster for a XDLJob object and makes changes based on the state read
// and what is in the XDLJob.Spec
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=training.kubedl.io,resources=xdljobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubedl.io,resources=xdljobs/status,verbs=get;update;patch

func (r *XDLJobReconciler) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Fetch the latest xdlJob instance.
	sharedXdlJob := &training.XDLJob{}
	err := r.ctrl.APIReader.Get(context.Background(), request.NamespacedName, sharedXdlJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to get job but it has been deleted", "key", request.String())
			r.ctrl.Metrics.DeletedInc()
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	xdlJob := sharedXdlJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(xdlJob, xdlJob.Spec.XDLReplicaSpecs)
	// No need to do reconcile or job has been deleted.
	if !needSync || xdlJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", xdlJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for xdl job.
	r.scheme.Default(xdlJob)

	result, err := r.ctrl.ReconcileJobs(xdlJob, xdlJob.Spec.XDLReplicaSpecs, xdlJob.Status, &xdlJob.Spec.RunPolicy, nil, nil)
	if err != nil {
		log.Error(err, "xdl job reconcile failed.")
		return result, err
	}
	return result, err
}

// SetupWithManager setup reconciler to the Manager with default RBAC. The Manager will set fields on the
// Controller and Start it when the Manager is Started.
func (r *XDLJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &training.XDLJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: job_controller.OnOwnerCreateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator, r.ctrl.Metrics),
		UpdateFunc: job_controller.OnOwnerUpdateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator),
		DeleteFunc: job_controller.OnOwnerDeleteFunc(r.ctrl, training.ExtractMetaFieldsFromObject, log),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.XDLJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.XDLJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	})
}

// ControllerName returns the Controller name
func (r *XDLJobReconciler) ControllerName() string {
	return controllerName
}

// GetAPIGroupVersionKind returns the GroupVersionKind of the API
func (r *XDLJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return training.SchemeGroupVersion.WithKind(training.XDLJobKind)
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *XDLJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return training.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *XDLJobReconciler) GetGroupNameLabelValue() string {
	return r.GetAPIGroupVersion().Group
}

// SetClusterSpec sets the cluster spec for the pod
func (r *XDLJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	xdlJob, ok := job.(*training.XDLJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XDLJob", job)
	}
	xdlConfigJson, err := genXDLConfigJSON(xdlJob, rtype, index)
	if err != nil {
		return err
	}

	if xdlConfigJson == "" {
		return nil
	}
	// add xdl environment variable
	for i := range podTemplate.Spec.Containers {
		container := &podTemplate.Spec.Containers[i]
		if len(container.Env) == 0 {
			container.Env = make([]corev1.EnvVar, 0)
		}
		for j := range container.Env {
			env := &container.Env[j]
			if env.Name == zkAddr {
				if strings.HasSuffix(env.Value, "/") {
					env.Value += string(xdlJob.UID)
				} else {
					env.Value += "/" + string(xdlJob.UID)
				}
			}
		}
		container.Env = append(container.Env,
			corev1.EnvVar{Name: taskType, Value: strings.ToLower(rtype)},
			corev1.EnvVar{Name: taskIndex, Value: index})
		if container.Name == r.GetDefaultContainerName() {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  xdlConfig,
				Value: xdlConfigJson,
			})
		}
	}
	return nil
}

// GetDefaultContainerName returns the default container name in pod
func (r *XDLJobReconciler) GetDefaultContainerName() string {
	return training.XDLJobDefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *XDLJobReconciler) GetDefaultContainerPortName() string {
	return training.XDLJobDefaultContainerPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *XDLJobReconciler) GetDefaultContainerPortNumber() int32 {
	return training.XDLJobDefaultPort
}

func (r *XDLJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		training.XDLReplicaTypePS,
		training.XDLReplicaTypeScheduler,
		training.XDLReplicaTypeWorker,
		training.XDLReplicaTypeExtendRole,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *XDLJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	// No master role in xdl job for now.
	return false
}

func (r *XDLJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	return ""
}
