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

package elasticdl

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/core"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/eventhandler"
	"github.com/alibaba/kubedl/pkg/metrics"
	utilruntime "github.com/alibaba/kubedl/pkg/util/runtime"
)

const (
	controllerName = "ElasticDLController"
)

var log = ctrl.Log.WithName("elasticdl-controller")

func NewReconciler(mgr ctrl.Manager, config options.JobControllerConfiguration) *ElasticDLJobReconciler {
	r := &ElasticDLJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(training.ElasticDLJobKind, r.Client), mgr.GetScheme())
	return r
}

var _ reconcile.Reconciler = &ElasticDLJobReconciler{}
var _ v1.ControllerInterface = &ElasticDLJobReconciler{}

// ElasticDLJobReconciler reconcile a ElastiDLJob object
type ElasticDLJobReconciler struct {
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
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=training.kubedl.io,resources=elasticdljobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubedl.io,resources=elasticdljobs/status,verbs=get;update;patch

func (r *ElasticDLJobReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch latest elasticdl job instance.
	sharedElasticDLJob := &training.ElasticDLJob{}
	err := r.ctrl.APIReader.Get(context.Background(), req.NamespacedName, sharedElasticDLJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to get job but it has been deleted", "key", req.String())
			r.ctrl.Metrics.DeletedInc()
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	elasticdlJob := sharedElasticDLJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(elasticdlJob, elasticdlJob.Spec.ElasticDLReplicaSpecs)
	// No need to do reconcile or job has been deleted.
	if !needSync || elasticdlJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", elasticdlJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for elasicdl job.
	r.scheme.Default(elasticdlJob)

	result, err := r.ctrl.ReconcileJobs(elasticdlJob, elasticdlJob.Spec.ElasticDLReplicaSpecs, elasticdlJob.Status, &elasticdlJob.Spec.RunPolicy, nil, nil)
	if err != nil {
		log.Error(err, "elasticdl job reconcile failed")
		return result, err
	}
	return result, nil
}

func (r *ElasticDLJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &training.ElasticDLJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: job_controller.OnOwnerCreateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator, r.ctrl.Metrics),
		UpdateFunc: job_controller.OnOwnerUpdateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator),
		DeleteFunc: job_controller.OnOwnerDeleteFunc(r.ctrl, training.ExtractMetaFieldsFromObject, log),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.ElasticDLJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.ElasticDLJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	})
}

func (r *ElasticDLJobReconciler) ControllerName() string {
	return controllerName
}

// GetAPIGroupVersionKind returns the GroupVersionKind of the API
func (r *ElasticDLJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return training.SchemeGroupVersion.WithKind(training.ElasticDLJobKind)
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *ElasticDLJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return training.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *ElasticDLJobReconciler) GetGroupNameLabelValue() string {
	return training.SchemeGroupVersion.Group
}

// GetDefaultContainerName returns the default container name in pod
func (r *ElasticDLJobReconciler) GetDefaultContainerName() string {
	return training.ElasticDLJobDefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *ElasticDLJobReconciler) GetDefaultContainerPortName() string {
	return training.ElasticDLJobDefaultPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *ElasticDLJobReconciler) GetDefaultContainerPortNumber() int32 {
	return training.ElasticDLJobDefaultPort
}

func (r *ElasticDLJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		training.ElasticDLReplicaTypeMaster,
	}
}

func (r *ElasticDLJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	_, ok := replicas[training.ElasticDLReplicaTypeMaster]
	return ok && rtype == training.ElasticDLReplicaTypeMaster
}

// SetClusterSpec sets the cluster spec for the pod
func (r *ElasticDLJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return nil
}

func (r *ElasticDLJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	return ""
}
