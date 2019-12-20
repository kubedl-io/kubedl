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

	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
func NewReconciler(mgr manager.Manager, config job_controller.JobControllerConfiguration) *XDLJobReconciler {
	r := &XDLJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	// Initialize pkg job controller with components we only need.
	r.ctrl = job_controller.JobController{
		Controller:       r,
		Expectations:     k8scontroller.NewControllerExpectations(),
		Config:           config,
		WorkQueue:        &commonutil.FakeWorkQueue{},
		Recorder:         r.recorder,
		MetricsCounter:   metrics.NewJobCounter(xdlv1alpha1.Kind, r.Client),
		MetricsHistogram: metrics.NewJobHistogram(xdlv1alpha1.Kind),
	}
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	return r
}

var _ reconcile.Reconciler = &XDLJobReconciler{}
var _ v1.ControllerInterface = &XDLJobReconciler{}

// XDLJobReconciler reconciles a XDLJob object
type XDLJobReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	ctrl     job_controller.JobController
}

// Reconcile reads that state of the cluster for a XDLJob object and makes changes based on the state read
// and what is in the XDLJob.Spec
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=xdl.alibaba.com,resources=xdljobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=xdl.alibaba.com,resources=xdljobs/status,verbs=get;update;patch
func (r *XDLJobReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the latest xdlJob instance.
	sharedXdlJob := &xdlv1alpha1.XDLJob{}
	err := r.Get(context.Background(), request.NamespacedName, sharedXdlJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to get job but it has been deleted", "key", request.String())
			if r.ctrl.MetricsCounter != nil {
				r.ctrl.MetricsCounter.DeletedInc()
			}
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	xdlJob := sharedXdlJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.satisfiedExpectations(xdlJob)
	// No need to do reconcile or job has been deleted.
	if !needSync || xdlJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", xdlJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for xdl job.
	r.scheme.Default(xdlJob)

	result, err := r.ctrl.ReconcileJobs(xdlJob, xdlJob.Spec.XDLReplicaSpecs, xdlJob.Status, &xdlJob.Spec.RunPolicy)
	if err != nil {
		log.Error(err, "xdl job reconcile failed.")
		return result, err
	}
	return result, err
}

// SetupWithManager setup reconciler to the Manager with default RBAC. The Manager will set fields on the
// Controller and Start it when the Manager is Started.
func (r *XDLJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &xdlv1alpha1.XDLJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: onOwnerCreateFunc(r),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &xdlv1alpha1.XDLJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &xdlv1alpha1.XDLJob{},
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
	return xdlv1alpha1.SchemeGroupVersionKind
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *XDLJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return xdlv1alpha1.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *XDLJobReconciler) GetGroupNameLabelValue() string {
	return xdlv1alpha1.GroupName
}

// SetClusterSpec sets the cluster spec for the pod
func (r *XDLJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	xdlJob, ok := job.(*xdlv1alpha1.XDLJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XDLJob", job)
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
	}
	return nil
}

// GetDefaultContainerName returns the default container name in pod
func (r *XDLJobReconciler) GetDefaultContainerName() string {
	return xdlv1alpha1.DefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *XDLJobReconciler) GetDefaultContainerPortName() string {
	return xdlv1alpha1.DefaultContainerPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *XDLJobReconciler) GetDefaultContainerPortNumber() int32 {
	return xdlv1alpha1.DefaultPort
}

func (r *XDLJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		xdlv1alpha1.XDLReplicaTypePS,
		xdlv1alpha1.XDLReplicaTypeScheduler,
		xdlv1alpha1.XDLReplicaTypeWorker,
		xdlv1alpha1.XDLReplicaTypeExtendRole,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *XDLJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	// No master role in xdl job for now.
	return false
}
