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

package pytorch

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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	pytorchv1 "github.com/alibaba/kubedl/api/pytorch/v1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
)

const (
	controllerName = "PytorchController"
)

var log = logf.Log.WithName("pytorch-controller")

func NewReconciler(mgr ctrl.Manager, config job_controller.JobControllerConfiguration) *PytorchJobReconciler {
	r := &PytorchJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(r, config, r.recorder, metrics.NewJobMetrics(pytorchv1.Kind, r.Client))
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	return r
}

var _ reconcile.Reconciler = &PytorchJobReconciler{}
var _ v1.ControllerInterface = &PytorchJobReconciler{}

// PytorchJobReconciler reconcile a PytorchJob object
type PytorchJobReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	ctrl     job_controller.JobController
}

// Reconcile reads that state of the cluster for a PytorchJob object and makes changes based on the state read
// and what is in the TFJob.Spec
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/status,verbs=get;update;patch

func (r *PytorchJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Fetch latest pytorch job instance.
	sharedPytorchJob := &pytorchv1.PyTorchJob{}
	err := r.Get(context.Background(), req.NamespacedName, sharedPytorchJob)
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

	pytorchJob := sharedPytorchJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(pytorchJob, pytorchJob.Spec.PyTorchReplicaSpecs)
	// No need to do reconcile or job has been deleted.
	if !needSync || pytorchJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", pytorchJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for pytorch job.
	r.scheme.Default(pytorchJob)

	result, err := r.ctrl.ReconcileJobs(pytorchJob, pytorchJob.Spec.PyTorchReplicaSpecs, pytorchJob.Status, &pytorchJob.Spec.RunPolicy)
	if err != nil {
		log.Error(err, "pytorch job reconcile failed")
		return result, err
	}
	return result, nil
}

func (r *PytorchJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &pytorchv1.PyTorchJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: onOwnerCreateFunc(r),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &pytorchv1.PyTorchJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &pytorchv1.PyTorchJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	})
}

func (r *PytorchJobReconciler) ControllerName() string {
	return controllerName
}

// GetAPIGroupVersionKind returns the GroupVersionKind of the API
func (r *PytorchJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return pytorchv1.SchemeGroupVersionKind
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *PytorchJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return pytorchv1.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *PytorchJobReconciler) GetGroupNameLabelValue() string {
	return pytorchv1.GroupName
}

// SetClusterSpec sets the cluster spec for the pod
func (r *PytorchJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	pytorchJob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PytorchJob", job)
	}
	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	masterPort, err := job_controller.GetPortFromJob(pytorchJob.Spec.PyTorchReplicaSpecs, pytorchv1.PyTorchReplicaTypeMaster, pytorchv1.DefaultContainerName, pytorchv1.DefaultPortName)
	if err != nil {
		return err
	}

	masterAddr := job_controller.GenGeneralName(pytorchJob.Name, strings.ToLower(string(pytorchv1.PyTorchReplicaTypeMaster)), strconv.Itoa(0))
	if rtype == strings.ToLower(string(pytorchv1.PyTorchReplicaTypeMaster)) {
		if rank != 0 {
			return fmt.Errorf("invalid config: There should be only a single master with index=0")
		}
		masterAddr = "localhost"
	} else {
		rank++
	}

	totalReplicas := int(k8sutil.GetTotalReplicas(pytorchJob.Spec.PyTorchReplicaSpecs))

	for i := range podTemplate.Spec.Containers {
		if len(podTemplate.Spec.Containers[i].Env) == 0 {
			podTemplate.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
		}
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_PORT",
			Value: strconv.Itoa(int(masterPort)),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_ADDR",
			Value: masterAddr,
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "WORLD_SIZE",
			Value: strconv.Itoa(totalReplicas),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "RANK",
			Value: strconv.Itoa(rank),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "0",
		})
	}
	return nil
}

// GetDefaultContainerName returns the default container name in pod
func (r *PytorchJobReconciler) GetDefaultContainerName() string {
	return pytorchv1.DefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *PytorchJobReconciler) GetDefaultContainerPortName() string {
	return pytorchv1.DefaultPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *PytorchJobReconciler) GetDefaultContainerPortNumber() int32 {
	return pytorchv1.DefaultPort
}

func (r *PytorchJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		pytorchv1.PyTorchReplicaTypeMaster,
		pytorchv1.PyTorchReplicaTypeWorker,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *PytorchJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	_, ok := replicas[pytorchv1.PyTorchReplicaTypeMaster]
	return ok && rtype == pytorchv1.PyTorchReplicaTypeMaster
}
