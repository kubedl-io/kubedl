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

	"github.com/openkruise/kruise/apis/apps/v1alpha1"
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

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/core"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/eventhandler"
	"github.com/alibaba/kubedl/pkg/metrics"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	patchutil "github.com/alibaba/kubedl/pkg/util/patch"
	"github.com/alibaba/kubedl/pkg/util/workloadgate"
)

const (
	controllerName = "PytorchController"
)

var log = logf.Log.WithName("pytorch-controller")

func NewReconciler(mgr ctrl.Manager, config options.JobControllerConfiguration) *PytorchJobReconciler {
	r := &PytorchJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(training.PyTorchJobKind, r.Client), mgr.GetScheme())
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	return r
}

var _ reconcile.Reconciler = &PytorchJobReconciler{}
var _ v1.ControllerInterface = &PytorchJobReconciler{}

// PytorchJobReconciler reconcile a PytorchJob object
type PytorchJobReconciler struct {
	client.Client
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
	ctrl        job_controller.JobController
	coordinator core.Coordinator
}

func (r *PytorchJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	for _, pod := range pods {
		rtype := pod.Labels[v1.ReplicaTypeLabel]
		rIndex, _ := strconv.Atoi(pod.Labels[v1.ReplicaIndexLabel])
		if rtype == strings.ToLower(string(training.PyTorchReplicaTypeMaster)) && rIndex == 0 {
			log.Info("select %s for model output", pod.Spec.NodeName)
			return pod.Spec.NodeName
		}
	}
	log.Info(fmt.Sprintf("no master replica type, select node %s for model output", pods[0].Spec.NodeName))
	return pods[0].Spec.NodeName
}

// Reconcile reads that state of the cluster for a PytorchJob object and makes changes based on the state read
// and what is in the TFJob.Spec
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=training.kubedl.io,resources=pytorchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubedl.io,resources=pytorchjobs/status,verbs=get;update;patch

func (r *PytorchJobReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch latest pytorch job instance.
	sharedPytorchJob := &training.PyTorchJob{}
	err := r.ctrl.APIReader.Get(context.Background(), req.NamespacedName, sharedPytorchJob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Cleanup preempt protector finalizers for elastic training jobs, however object
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

	pytorchJob := sharedPytorchJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(pytorchJob, pytorchJob.Spec.PyTorchReplicaSpecs) || r.EnableElasticScaling(pytorchJob, &pytorchJob.Spec.RunPolicy)
	// No need to do reconcile or job has been deleted.
	if !needSync || pytorchJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", pytorchJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// Set default properties for pytorch job.
	r.scheme.Default(pytorchJob)

	result, err := r.ctrl.ReconcileJobs(pytorchJob, pytorchJob.Spec.PyTorchReplicaSpecs, pytorchJob.Status, &pytorchJob.Spec.RunPolicy, pytorchJob.Spec.ModelVersion, pytorchJob.Spec.CacheBackend)
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
	if err = c.Watch(&source.Kind{Type: &training.PyTorchJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: job_controller.OnOwnerCreateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator, r.ctrl.Metrics),
		UpdateFunc: job_controller.OnOwnerUpdateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator),
		DeleteFunc: job_controller.OnOwnerDeleteFunc(r.ctrl, training.ExtractMetaFieldsFromObject, log),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.PyTorchJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.PyTorchJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnServiceCreateFunc,
		UpdateFunc: r.ctrl.OnServiceUpdateFunc,
		DeleteFunc: r.ctrl.OnServiceDeleteFunc,
	}); err != nil {
		return err
	}

	if _, enabled := workloadgate.IsWorkloadEnable(&v1alpha1.ContainerRecreateRequest{}, mgr.GetScheme()); enabled {
		if err = c.Watch(&source.Kind{Type: &v1alpha1.ContainerRecreateRequest{}}, &handler.EnqueueRequestForOwner{
			OwnerType:    &training.PyTorchJob{},
			IsController: false,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *PytorchJobReconciler) ControllerName() string {
	return controllerName
}

// GetAPIGroupVersionKind returns the GroupVersionKind of the API
func (r *PytorchJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return training.SchemeGroupVersion.WithKind(training.PyTorchJobKind)
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *PytorchJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return training.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *PytorchJobReconciler) GetGroupNameLabelValue() string {
	return training.SchemeGroupVersion.Group
}

// SetClusterSpec sets the cluster spec for the pod
func (r *PytorchJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	pytorchJob, ok := job.(*training.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PytorchJob", job)
	}
	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	masterPort, err := job_controller.GetPortFromJob(pytorchJob.Spec.PyTorchReplicaSpecs, training.PyTorchReplicaTypeMaster, training.PyTorchJobDefaultContainerName, training.PyTorchJobDefaultPortName)
	if err != nil {
		return err
	}

	masterRole := rtype == strings.ToLower(string(training.PyTorchReplicaTypeMaster))
	if masterHostPort, ok := job_controller.GetHostNetworkPortFromContext(ctx, "master", "0"); job_controller.EnableHostNetwork(pytorchJob) && ok {
		if masterRole || features.KubeDLFeatureGates.Enabled(features.HostNetWithHeadlessSvc) {
			masterPort = masterHostPort
		}
	}

	masterAddr := commonutil.GenGeneralName(pytorchJob.Name, strings.ToLower(string(training.PyTorchReplicaTypeMaster)), strconv.Itoa(0))
	if masterRole {
		if rank != 0 {
			return fmt.Errorf("invalid config: There should be only a single master with index=0")
		}
		if features.KubeDLFeatureGates.Enabled(features.PyTorchLocalMasterAddr) {
			masterAddr = "localhost"
		}
	} else {
		rank++
	}

	if job_controller.EnableErrorMonitoring(pytorchJob) {
		podTemplate.Finalizers = append(podTemplate.Finalizers, v1.FinalizerPreemptProtector)
	}

	totalReplicas := int(k8sutil.GetTotalExcludedReplicas(pytorchJob.Spec.PyTorchReplicaSpecs, v1.JobReplicaTypeAIMaster))
	enableElasticScaling := pytorchJob.Annotations[v1.AnnotationEnableElasticTraining] == "true"
	if enableElasticScaling && !masterRole && rtype != "aimaster" {
		AddImageWarmupForWorker(podTemplate, r.GetDefaultContainerName())
		err = AddMasterWaiterForWorker(podTemplate, InitContainerParam{
			MasterAddr:         masterAddr,
			InitContainerImage: "docker.io/alpine:3.10",
		})
		if err != nil {
			return err
		}
	}

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
			Name:  "RANK",
			Value: strconv.Itoa(rank),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "0",
		})
		if enableElasticScaling && rtype != "aimaster" {
			// Job enables elastic scaling select value of AnnotationWorldSize as its
			// WORLD_SIZE env value via field-path, the annotated value will be mutated
			// during scaling progress and processes in container will acknowledge the
			// updated world-size value after restarted.
			if podTemplate.Annotations == nil {
				podTemplate.Annotations = make(map[string]string)
			}
			podTemplate.Annotations[AnnotationWorldSize] = strconv.Itoa(totalReplicas)

			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
				Name: "WORLD_SIZE",
				ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: fmt.Sprintf("metadata.annotations['%s']", AnnotationWorldSize),
				}},
			})

			// Overwrite restart policy as OnFailure so that container will be started again
			// by kubelet after kruise daemonset forcefully execute restarting container through
			// CRI calls bypasses kubelet.
			podTemplate.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		} else {
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "WORLD_SIZE",
				Value: strconv.Itoa(totalReplicas),
			})
		}
	}
	return nil
}

// GetDefaultContainerName returns the default container name in pod
func (r *PytorchJobReconciler) GetDefaultContainerName() string {
	return training.PyTorchJobDefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *PytorchJobReconciler) GetDefaultContainerPortName() string {
	return training.PyTorchJobDefaultPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *PytorchJobReconciler) GetDefaultContainerPortNumber() int32 {
	return training.PyTorchJobDefaultPort
}

func (r *PytorchJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	return []v1.ReplicaType{
		v1.JobReplicaTypeAIMaster,
		training.PyTorchReplicaTypeMaster,
		training.PyTorchReplicaTypeWorker,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *PytorchJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	_, ok := replicas[training.PyTorchReplicaTypeMaster]
	return ok && rtype == training.PyTorchReplicaTypeMaster
}

func (r *PytorchJobReconciler) cleanUpPreemptFinalizers(namespace, name string) error {
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
