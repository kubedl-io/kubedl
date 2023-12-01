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

// Package tensorflow provides a Kubernetes controller for a TFJob resource.
package tensorflow

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

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/core"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/eventhandler"
	"github.com/alibaba/kubedl/pkg/metrics"
	"github.com/alibaba/kubedl/pkg/tensorboard"
	utilruntime "github.com/alibaba/kubedl/pkg/util/runtime"
)

const (
	controllerName = "TensorFlowController"
)

const (
	// tfConfig is the environment variable name of TensorFlow cluster spec.
	tfConfig = "TF_CONFIG"
)

var log = logf.Log.WithName("tf-controller")

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr ctrl.Manager, config options.JobControllerConfiguration) *TFJobReconciler {
	r := &TFJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(training.TFJobKind, r.Client), mgr.GetScheme())
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	return r
}

var _ reconcile.Reconciler = &TFJobReconciler{}
var _ v1.ControllerInterface = &TFJobReconciler{}

// TFJobReconciler reconciles a TFJob object
type TFJobReconciler struct {
	client.Client
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
	coordinator core.Coordinator
	ctrl        job_controller.JobController
	utilruntime.EmptyScaleImpl
}

func (r *TFJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) string {
	var chiefPod *corev1.Pod
	var masterPod *corev1.Pod
	var worker0Pod *corev1.Pod

	for _, pod := range pods {
		rtype := pod.Labels[v1.ReplicaTypeLabel]
		rIndex, _ := strconv.Atoi(pod.Labels[v1.ReplicaIndexLabel])
		if rtype == strings.ToLower(string(training.TFReplicaTypeMaster)) && rIndex == 0 {
			masterPod = pod
		}
		if rtype == strings.ToLower(string(training.TFReplicaTypeChief)) && rIndex == 0 {
			chiefPod = pod
		}
		if rtype == strings.ToLower(string(training.TFReplicaTypeWorker)) && rIndex == 0 {
			worker0Pod = pod
		}
	}
	nodeName := ""
	if chiefPod != nil {
		nodeName = chiefPod.Spec.NodeName
	}
	if masterPod != nil {
		nodeName = masterPod.Spec.NodeName
	}
	if worker0Pod != nil {
		nodeName = worker0Pod.Spec.NodeName
	}
	if nodeName != "" {
		log.Info(fmt.Sprintf("select %s for model output", nodeName))
		return nodeName
	} else {
		log.Info("no node selected for model output")
		return nodeName
	}
}

// Reconcile reads that state of the cluster for a XDLJob object and makes changes based on the state read
// and what is in the TFJob.Spec
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;create;patch;
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=training.kubedl.io,resources=tfjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubedl.io,resources=tfjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Note, this is added for kubedl-dashboard to list nodes, not for kubedl-controller
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list

func (r *TFJobReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the TFJob tfJob
	sharedTfJob := &training.TFJob{}
	err := r.ctrl.APIReader.Get(context.Background(), req.NamespacedName, sharedTfJob)
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

	tfJob := sharedTfJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(tfJob, tfJob.Spec.TFReplicaSpecs)
	// No need to do reconcile or job has been deleted.
	if !needSync || tfJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", tfJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for tensorflow job.
	r.scheme.Default(tfJob)

	result, err := r.ctrl.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy, tfJob.Spec.ModelVersion, tfJob.Spec.CacheBackend)
	if err != nil {
		log.Error(err, "tensorflow job reconcile failed")
		return result, err
	}

	// sync tensorboard using chief or worker spec.
	masterType, masterSpec := r.getMasterSpec(tfJob.Spec.TFReplicaSpecs)
	if err := tensorboard.ReconcileTensorBoard(r.ctrl, r.Client, tfJob,
		masterType, masterSpec, tfJob.Status, &result); err != nil {
		log.Error(err, "ReconcileTensorBoard error %v")
		return result, err
	}
	return result, err
}

// SetupWithManager setup reconciler to the Manager with default RBAC. The Manager will set fields on the
// Controller and Start it when the Manager is Started.
func (r *TFJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &training.TFJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: job_controller.OnOwnerCreateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator, r.ctrl.Metrics),
		UpdateFunc: job_controller.OnOwnerUpdateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator),
		DeleteFunc: job_controller.OnOwnerDeleteFunc(r.ctrl, training.ExtractMetaFieldsFromObject, log),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.TFJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.TFJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	})
}

// ControllerName returns the Controller name
func (r *TFJobReconciler) ControllerName() string {
	return controllerName
}

// GetAPIGroupVersionKind returns the GroupVersionKind of the API
func (r *TFJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return training.SchemeGroupVersion.WithKind(training.TFJobKind)
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *TFJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return training.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *TFJobReconciler) GetGroupNameLabelValue() string {
	return training.SchemeGroupVersion.Group
}

// SetClusterSpec generates and sets TF_CONFIG for the given podTemplateSpec.
func (r *TFJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplateSpec *corev1.PodTemplateSpec, rt, index string) error {
	tfJob, ok := job.(*training.TFJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of TFJob", job)
	}

	// Do not set TF_CONFIG for local training jobs.
	if !isDistributed(tfJob) {
		return nil
	}
	// Generate TF_CONFIG JSON string.
	tfConfigStr, err := genTFConfigJSONStr(ctx, tfJob, rt, index)
	if err != nil {
		return err
	}

	if tfConfigStr == "" {
		return nil
	}
	// Add TF_CONFIG environment variable to tensorflow container in the pod.
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name == training.TFJobDefaultContainerName {
			if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
				podTemplateSpec.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
			}
			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  tfConfig,
				Value: tfConfigStr,
			})
			break
		}
	}
	return nil
}

// isDistributed returns if the TFJob is a distributed training job.
// Ref https://github.com/kubeflow/tf-operator/issues/1078.
func isDistributed(tfJob *training.TFJob) bool {
	replicas := tfJob.Spec.TFReplicaSpecs
	distributionCount := 0
	allTypes := []v1.ReplicaType{
		training.TFReplicaTypeChief,
		training.TFReplicaTypeEval,
		training.TFReplicaTypeMaster,
		training.TFReplicaTypePS,
		training.TFReplicaTypeWorker,
	}
	// Check if there is only one replica.
	for _, typ := range allTypes {
		if replicas[typ] != nil {
			if replicas[typ].Replicas == nil {
				distributionCount++
			} else {
				distributionCount += int(*replicas[typ].Replicas)
			}
		}
	}
	return distributionCount != 1
}

// GetDefaultContainerName returns the default container name in pod
func (r *TFJobReconciler) GetDefaultContainerName() string {
	return training.TFJobDefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *TFJobReconciler) GetDefaultContainerPortName() string {
	return training.TFJobDefaultPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *TFJobReconciler) GetDefaultContainerPortNumber() int32 {
	return training.TFJobDefaultPort
}

// Get replicas reconcile orders so that replica type with higher priority can be created earlier.
func (r *TFJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		training.TFReplicaTypePS,
		training.TFReplicaTypeMaster,
		training.TFReplicaTypeChief,
		training.TFReplicaTypeWorker,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *TFJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	return training.IsTFJobChieforMaster(rtype)
}

// getMasterSpec returns chief or worker spec.
func (r *TFJobReconciler) getMasterSpec(replicas map[v1.ReplicaType]*v1.ReplicaSpec) (v1.ReplicaType, *v1.ReplicaSpec) {
	if spec, ok := replicas[training.TFReplicaTypeChief]; ok {
		return training.TFReplicaTypeChief, spec
	}
	if spec, ok := replicas[training.TFReplicaTypeMaster]; ok {
		return training.TFReplicaTypeMaster, spec
	}
	return training.TFReplicaTypeWorker, replicas[training.TFReplicaTypeWorker]
}
