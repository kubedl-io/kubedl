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
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	"github.com/alibaba/kubedl/pkg/gang_schedule"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
	"github.com/alibaba/kubedl/pkg/util"
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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
func NewReconciler(mgr ctrl.Manager, config job_controller.JobControllerConfiguration) *TFJobReconciler {
	r := &TFJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	// Initialize pkg job controller with components we only need.
	r.ctrl = job_controller.JobController{
		Controller:     r,
		Expectations:   k8scontroller.NewControllerExpectations(),
		Config:         config,
		WorkQueue:      &util.FakeWorkQueue{},
		Recorder:       r.recorder,
		MetricsCounter: metrics.NewJobCounter("tf", metrics.TFJobRunningCounter(r.Client)),
		MetricsGauge:   metrics.NewJobGauge("tf"),
	}
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = gang_schedule.Get(r.ctrl.Config.GangSchedulerName)
	}
	return r
}

var _ reconcile.Reconciler = &TFJobReconciler{}
var _ v1.ControllerInterface = &TFJobReconciler{}

// TFJobReconciler reconciles a TFJob object
type TFJobReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	ctrl     job_controller.JobController
}

// Reconcile reads that state of the cluster for a XDLJob object and makes changes based on the state read
// and what is in the TFJob.Spec
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs/status,verbs=get;update;patch
func (r *TFJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Fetch the TFJob tfJob
	sharedTfJob := &tfv1.TFJob{}
	err := r.Get(context.Background(), req.NamespacedName, sharedTfJob)
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
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	tfJob := sharedTfJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.satisfiedExpectations(tfJob)
	// No need to do reconcile or job has been deleted.
	if !needSync || tfJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", tfJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for tensorflow job.
	r.scheme.Default(tfJob)

	result, err := r.ctrl.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy)
	if err != nil {
		log.Error(err, "tensorflow job reconcile failed")
		return result, err
	}
	return result, err
}

// SetupWithManager setup reconciler to the Manager with default RBAC. The Manager will set fields on the
// Controller and Start it when the Manager is Started.
func (r *TFJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &tfv1.TFJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: onOwnerCreateFunc(r),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &tfv1.TFJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &tfv1.TFJob{},
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
	return tfv1.SchemeGroupVersionKind
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *TFJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return tfv1.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *TFJobReconciler) GetGroupNameLabelValue() string {
	return tfv1.GroupName
}

// SetClusterSpec generates and sets TF_CONFIG for the given podTemplateSpec.
func (r *TFJobReconciler) SetClusterSpec(job interface{}, podTemplateSpec *corev1.PodTemplateSpec, rt, index string) error {
	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of TFJob", job)
	}

	// Do not set TF_CONFIG for local training jobs.
	if !isDistributed(tfJob) {
		return nil
	}
	// Generate TF_CONFIG JSON string.
	tfConfigStr, err := genTFConfigJSONStr(tfJob, rt, index)
	if err != nil {
		return err
	}

	if tfConfigStr == "" {
		return nil
	}
	// Add TF_CONFIG environment variable to tensorflow container in the pod.
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name == tfv1.DefaultContainerName {
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
func isDistributed(tfJob *tfv1.TFJob) bool {
	replicas := tfJob.Spec.TFReplicaSpecs
	distributionCount := 0
	allTypes := []v1.ReplicaType{
		tfv1.TFReplicaTypeChief,
		tfv1.TFReplicaTypeEval,
		tfv1.TFReplicaTypeMaster,
		tfv1.TFReplicaTypePS,
		tfv1.TFReplicaTypeWorker,
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
	return tfv1.DefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *TFJobReconciler) GetDefaultContainerPortName() string {
	return tfv1.DefaultPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *TFJobReconciler) GetDefaultContainerPortNumber() int32 {
	return tfv1.DefaultPort
}

// Get replicas reconcile orders so that replica type with higher priority can be created earlier.
func (r *TFJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		tfv1.TFReplicaTypePS,
		tfv1.TFReplicaTypeMaster,
		tfv1.TFReplicaTypeChief,
		tfv1.TFReplicaTypeWorker,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *TFJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	return tfv1.IsChieforMaster(rtype)
}
