/*
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

package xgboostjob

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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
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
	controllerName = "XGBoostController"
)

var log = logf.Log.WithName("xgb-controller")

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager, config options.JobControllerConfiguration) *XgboostJobReconciler {
	r := &XgboostJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(v1alpha1.XGBoostJobKind, r.Client), mgr.GetScheme())
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	return r
}

var _ reconcile.Reconciler = &XgboostJobReconciler{}
var _ v1.ControllerInterface = &XgboostJobReconciler{}

// XgboostJobReconciler reconciles a XGBoostJob object
type XgboostJobReconciler struct {
	client.Client
	scheme      *runtime.Scheme
	ctrl        job_controller.JobController
	recorder    record.EventRecorder
	coordinator core.Coordinator
	utilruntime.EmptyScaleImpl
}

// Reconcile reads that state of the cluster for a XGBoostJob object and makes changes based on the state read
// and what is in the XGBoostJob.Spec
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=training.kubedl.io,resources=xgboostjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubedl.io,resources=xgboostjobs/status,verbs=get;update;patch

func (r *XgboostJobReconciler) Reconcile(_ context.Context, req reconcile.Request) (reconcile.Result, error) {
	// Fetch the XGBoostJob instance
	xgboostjob := &v1alpha1.XGBoostJob{}
	err := r.ctrl.APIReader.Get(context.Background(), req.NamespacedName, xgboostjob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to get job but it has been deleted", "key", req.String())
			r.ctrl.Metrics.DeletedInc()
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the req.
		return reconcile.Result{}, err
	}

	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(xgboostjob, xgboostjob.Spec.XGBReplicaSpecs)

	if !needSync || xgboostjob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", xgboostjob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for xgboost job
	r.scheme.Default(xgboostjob)

	result, err := r.ctrl.ReconcileJobs(xgboostjob, xgboostjob.Spec.XGBReplicaSpecs, xgboostjob.Status.JobStatus, &xgboostjob.Spec.RunPolicy, nil, nil)
	if err != nil {
		log.Error(err, "xgboost job reconcile failed")
		return result, err
	}
	return result, err
}

// SetupWithManager setup reconciler to the Manager with default RBAC. The Manager will set fields on the
// Controller and Start it when the Manager is Started.
func (r *XgboostJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &v1alpha1.XGBoostJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: job_controller.OnOwnerCreateFunc(r.scheme, v1alpha1.ExtractMetaFieldsFromObject, log, r.coordinator, r.ctrl.Metrics),
		UpdateFunc: job_controller.OnOwnerUpdateFunc(r.scheme, v1alpha1.ExtractMetaFieldsFromObject, log, r.coordinator),
		DeleteFunc: job_controller.OnOwnerDeleteFunc(r.ctrl, v1alpha1.ExtractMetaFieldsFromObject, log),
	}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &v1alpha1.XGBoostJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &v1alpha1.XGBoostJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	})
}

func (r *XgboostJobReconciler) ControllerName() string {
	return controllerName
}

func (r *XgboostJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return v1alpha1.SchemeGroupVersion.WithKind(v1alpha1.XGBoostJobKind)
}

func (r *XgboostJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return v1alpha1.SchemeGroupVersion
}

func (r *XgboostJobReconciler) GetGroupNameLabelValue() string {
	return v1alpha1.SchemeGroupVersion.Group
}

func (r *XgboostJobReconciler) GetDefaultContainerName() string {
	return v1alpha1.XGBoostJobDefaultContainerName
}

func (r *XgboostJobReconciler) GetDefaultContainerPortNumber() int32 {
	return v1alpha1.XGBoostJobDefaultPort
}

func (r *XgboostJobReconciler) GetDefaultContainerPortName() string {
	return v1alpha1.XGBoostJobDefaultContainerPortName
}

func (r *XgboostJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec,
	rtype v1.ReplicaType, index int) bool {
	_, ok := replicas[v1alpha1.XGBoostReplicaTypeMaster]
	return ok && rtype == v1alpha1.XGBoostReplicaTypeMaster
}

func (r *XgboostJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		v1alpha1.XGBoostReplicaTypeMaster,
		v1alpha1.XGBoostReplicaTypeWorker,
	}
}

// SetClusterSpec sets the cluster spec for the pod
func (r *XgboostJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, index)
}

func (r *XgboostJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	return ""
}
