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
	"flag"
	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
	"github.com/alibaba/kubedl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/apis/apps"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
	"path/filepath"
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
	controllerName = "XGBoostController"
	// gang scheduler name.
	gangSchedulerName = "kube-batch"
)

var log = logf.Log.WithName("controller")

const RecommendedKubeConfigPathEnv = "KUBECONFIG"

// newReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr manager.Manager, config job_controller.JobControllerConfiguration) *XgboostJobReconciler {
	r := &XgboostJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())

	var mode string
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig_", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig_", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	mode = flag.Lookup("mode").Value.(flag.Getter).Get().(string)
	if mode == "local" {
		log.Info("Running controller in local mode, using kubeconfig file")
		/// TODO, add the master url and kubeconfigpath with user input
		kcfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Info("Error building kubeconfig: %s", err.Error())
			panic(err.Error())
		}
		_ = kcfg
	} else if mode == "in-cluster" {
		log.Info("Running controller in in-cluster mode")
		/// TODO, add the master url and kubeconfigpath with user input
		kcfg, err := rest.InClusterConfig()
		if err != nil {
			log.Info("Error getting in-cluster kubeconfig")
			panic(err.Error())
		}
		_ = kcfg
	} else {
		log.Info("Given mode is not valid: ", "mode", mode)
		panic("-mode should be either local or in-cluster")
	}

	// Initialize pkg job controller with components we only need.
	r.ctrl = job_controller.JobController{
		Controller:     r,
		Expectations:   k8scontroller.NewControllerExpectations(),
		Config:         config,
		WorkQueue:      &util.FakeWorkQueue{},
		Recorder:       r.recorder,
		Client:         r.Client,
		MetricsCounter: metrics.NewJobCounter("xgboost", metrics.XGBoostJobRunningCounter(r.Client)),
		MetricsGauge:   metrics.NewJobGauge("xgboost"),
	}

	return r
}

var _ reconcile.Reconciler = &XgboostJobReconciler{}
var _ v1.ControllerInterface = &XgboostJobReconciler{}

// XgboostJobReconciler reconciles a XGBoostJob object
type XgboostJobReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	ctrl     job_controller.JobController
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a XGBoostJob object and makes changes based on the state read
// and what is in the XGBoostJob.Spec
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=xgboostjob.kubeflow.org,resources=xgboostjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=xgboostjob.kubeflow.org,resources=xgboostjobs/status,verbs=get;update;patch
func (r *XgboostJobReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	// Fetch the XGBoostJob instance
	xgboostjob := &v1alpha1.XGBoostJob{}
	err := r.Get(context.Background(), req.NamespacedName, xgboostjob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to get job but it has been deleted", "key", req.String())
			if r.ctrl.MetricsCounter != nil {
				r.ctrl.MetricsCounter.DeletedInc()
				r.ctrl.MetricsCounter.RunningGauge()
			}
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the req.
		return reconcile.Result{}, err
	}

	// Check reconcile is required.
	needSync := r.satisfiedExpectations(xgboostjob)

	if !needSync || xgboostjob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", xgboostjob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for xgboost job
	r.scheme.Default(xgboostjob)

	result, err := r.ctrl.ReconcileJobs(xgboostjob, xgboostjob.Spec.XGBReplicaSpecs, xgboostjob.Status.JobStatus, &xgboostjob.Spec.RunPolicy)
	if err != nil {
		log.Error(err, "xgboost job reconcile failed")
		return result, err
	}
	return result, err
}

// SetupWithManager setup reconciler to the Manager with default RBAC. The Manager will set fields on the
// Controller and Start it when the Manager is Started.
func (r *XgboostJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &v1alpha1.XGBoostJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: onOwnerCreateFunc(r),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &apps.Deployment{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &v1alpha1.XGBoostJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: onDependentCreateFunc(r),
		DeleteFunc: onDependentDeleteFunc(r),
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
	return v1alpha1.SchemeGroupVersionKind
}

func (r *XgboostJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return v1alpha1.SchemeGroupVersion
}

func (r *XgboostJobReconciler) GetGroupNameLabelValue() string {
	return v1alpha1.GroupName
}

func (r *XgboostJobReconciler) GetDefaultContainerName() string {
	return v1alpha1.DefaultContainerName
}

func (r *XgboostJobReconciler) GetDefaultContainerPortNumber() int32 {
	return v1alpha1.DefaultPort
}

func (r *XgboostJobReconciler) GetDefaultContainerPortName() string {
	return v1alpha1.DefaultContainerPortName
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
func (r *XgboostJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, index)
}
