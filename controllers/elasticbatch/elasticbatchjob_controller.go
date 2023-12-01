/*
Copyright 2022 The Alibaba Authors.

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

package elasticbatch

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	inference "github.com/alibaba/kubedl/apis/inference/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/core"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/eventhandler"
	"github.com/alibaba/kubedl/pkg/metrics"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	patchutil "github.com/alibaba/kubedl/pkg/util/patch"
	utilruntime "github.com/alibaba/kubedl/pkg/util/runtime"
)

const (
	controllerName = "ElasticBatchController"

	// elasticBatchConfig is the environment variable name of ElasticBatch cluster spec.
	elasticBatchConfig = "ElASTICBATCH_CONFIG"
)

var log = logf.Log.WithName("elasticbatch-controller")

func NewReconciler(mgr ctrl.Manager, config options.JobControllerConfiguration) *ElasticBatchJobReconciler {
	r := &ElasticBatchJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(inference.ElasticBatchJobKind, r.Client), mgr.GetScheme())
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	return r
}

var _ reconcile.Reconciler = &ElasticBatchJobReconciler{}
var _ v1.ControllerInterface = &ElasticBatchJobReconciler{}

// ElasticBatchJobReconciler reconciles a ElasticBatchJob object
type ElasticBatchJobReconciler struct {
	client.Client
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
	ctrl        job_controller.JobController
	coordinator core.Coordinator
	utilruntime.EmptyScaleImpl
}

// Reconcile reads that state of the cluster for a ElasticBatchJob object and makes changes based on the state read
// and what is in the ElasticBatchJob.Spec
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=inference.kubedl.io,resources=elasticbatchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inference.kubedl.io,resources=elasticbatchjobs/status,verbs=get;update;patch

// Note, this is added for kubedl-dashboard to list nodes, not for kubedl-controller
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list

func (r *ElasticBatchJobReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch latest elasticbatch job instance.
	sharedElasticBatchJob := &inference.ElasticBatchJob{}
	err := r.ctrl.APIReader.Get(context.Background(), req.NamespacedName, sharedElasticBatchJob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Cleanup preempt protector finalizers for elastic inference jobs, however object
			// has completely deleted, and we'd clean up finalizers to sweep remaining pods.
			if err = r.cleanUpPreemptFinalizers(req.Namespace, req.Name); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("try to get job but it has been deleted", "key", req.String())
			r.ctrl.Metrics.DeletedInc()
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	elasticbatchJob := sharedElasticBatchJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(elasticbatchJob, elasticbatchJob.Spec.ElasticBatchReplicaSpecs) || r.EnableElasticScaling(elasticbatchJob, &elasticbatchJob.Spec.RunPolicy)
	// No need to do reconcile or job has been deleted.
	if !needSync || elasticbatchJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", elasticbatchJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for elasticbatch job.
	r.scheme.Default(elasticbatchJob)

	result, err := r.ctrl.ReconcileJobs(elasticbatchJob, elasticbatchJob.Spec.ElasticBatchReplicaSpecs, elasticbatchJob.Status, &elasticbatchJob.Spec.RunPolicy, nil, nil)
	if err != nil {
		log.Error(err, "elasticbatch job reconcile failed")
		return result, err
	}
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElasticBatchJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &inference.ElasticBatchJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: onOwnerCreateFunc(r),
		DeleteFunc: OnOwnerDeleteAndDeletionExpectationFunc(r.ctrl),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &inference.ElasticBatchJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &inference.ElasticBatchJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	})
}

// ControllerName returns the Controller name
func (r *ElasticBatchJobReconciler) ControllerName() string {
	return controllerName
}

// GetAPIGroupVersionKind returns the GroupVersionKind of the API
func (r *ElasticBatchJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return inference.GroupVersion.WithKind(inference.ElasticBatchJobKind)
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *ElasticBatchJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return inference.GroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *ElasticBatchJobReconciler) GetGroupNameLabelValue() string {
	return inference.GroupVersion.Group
}

// SetClusterSpec generates and sets ElASTICBATCH_CONFIG for the given podTemplateSpec.
func (r *ElasticBatchJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplateSpec *corev1.PodTemplateSpec, rtype, index string) error {
	elasticBatchJob, ok := job.(*inference.ElasticBatchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of ElasticBatchJob", job)
	}
	if strings.EqualFold(rtype, string(v1.JobReplicaTypeAIMaster)) {
		return nil
	}

	if job_controller.EnableErrorMonitoring(elasticBatchJob) {
		podTemplateSpec.Finalizers = append(podTemplateSpec.Finalizers, v1.FinalizerPreemptProtector)
	}

	// Generate ElASTICBATCH_CONFIG JSON string.
	elasticBatchConfigStr, err := genElasticBatchConfigJSONStr(ctx, elasticBatchJob, strings.ToLower(rtype), index)
	if err != nil {
		return err
	}

	if elasticBatchConfigStr == "" {
		return nil
	}

	// Add ElasticBatchConfig environment variable to elasticbatch container in the pod.
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name == inference.ElasticBatchJobDefaultContainerName {
			if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
				podTemplateSpec.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
			}
			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  elasticBatchConfig,
				Value: elasticBatchConfigStr,
			})
			break
		}
	}
	return nil
}

// GetDefaultContainerName returns the default container name in pod
func (r *ElasticBatchJobReconciler) GetDefaultContainerName() string {
	return inference.ElasticBatchJobDefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *ElasticBatchJobReconciler) GetDefaultContainerPortName() string {
	return inference.ElasticBatchJobDefaultPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *ElasticBatchJobReconciler) GetDefaultContainerPortNumber() int32 {
	return inference.ElasticBatchJobDefaultPort
}

// Get replicas reconcile orders so that replica type with higher priority can be created earlier.
func (r *ElasticBatchJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		inference.ElasticBatchReplicaTypeAIMaster,
		inference.ElasticBatchReplicaTypeWorker,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *ElasticBatchJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	// No master role in elasticbatch job for now.
	return false
}

func (r *ElasticBatchJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	return ""
}

func (r *ElasticBatchJobReconciler) cleanUpPreemptFinalizers(namespace, name string) error {
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(name),
	})
	pods := corev1.PodList{}
	if err := r.Client.List(context.Background(), &pods, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return err
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if k8sutil.HasFinalizer(pod.Finalizers, v1.FinalizerPreemptProtector) {
			klog.V(2).Infof("pod %s has finalizer %s, need to remove", pod.Name, v1.FinalizerPreemptProtector)
			patch := patchutil.NewStrategicPatch()
			patch.RemoveFinalizer(v1.FinalizerPreemptProtector)
			if err := r.Client.Patch(context.Background(), pod, patch); err != nil {
				return err
			}
		}
	}
	return nil
}
