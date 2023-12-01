/*
Copyright 2021 The Alibaba Authors.

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

package mpi

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

	"github.com/spf13/pflag"

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

func init() {
	pflag.StringVar(&kubectlDeliveryImage, "kubectl-delivery-image", defaultKubectlDeliveryImage, "utility image to delivery kubectl binary")
	pflag.BoolVar(&launcherRunsWorkload, "launcher-runs-workloads", true, "Set launcher run the workload when launcher has GPU")
}

const (
	controllerName              = "MPIController"
	defaultKubectlDeliveryImage = "kubedl/kubectl-delivery:latest"
	initContainerCpu            = "100m"
	initContainerMem            = "128Mi"
)

var (
	log = logf.Log.WithName("mpi-controller")

	kubectlDeliveryImage string
	launcherRunsWorkload bool
)

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr ctrl.Manager, config options.JobControllerConfiguration) *MPIJobReconciler {
	r := &MPIJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
	r.ctrl = job_controller.NewJobController(mgr, r, config, r.recorder, metrics.NewJobMetrics(training.MPIJobKind, r.Client), mgr.GetScheme())
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) {
		r.coordinator = core.NewCoordinator(mgr)
	}
	return r
}

var _ reconcile.Reconciler = &MPIJobReconciler{}
var _ v1.ControllerInterface = &MPIJobReconciler{}

// MPIJobReconciler reconciles a MPIJob object.
type MPIJobReconciler struct {
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
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups=scheduling.incubator.k8s.io;scheduling.sigs.dev;scheduling.volcano.sh,resources=podgroups;queues,verbs=*
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=*
// +kubebuilder:rbac:groups=training.kubedl.io,resources=mpijobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=training.kubedl.io,resources=mpijobs/status,verbs=get;update;patch

// Reconcile reads that state of the cluster for a MPIJob object and makes changes based on the state read
// and what is in the MPIJob.Spec
func (r *MPIJobReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		result ctrl.Result
		err    error
	)

	// Fetch the latest mpi job.
	sharedMPIJob := &training.MPIJob{}
	err = r.ctrl.APIReader.Get(context.Background(), req.NamespacedName, sharedMPIJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to get job but it has been deleted", "key", req.String())
			r.ctrl.Metrics.DeletedInc()
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	mpiJob := sharedMPIJob.DeepCopy()
	// Check reconcile is required.
	needSync := r.ctrl.SatisfyExpectations(mpiJob, mpiJob.Spec.MPIReplicaSpecs)
	// No need to do reconcile or job has been deleted.
	if !needSync || mpiJob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"job namespace", mpiJob.Namespace, "job name", mpiJob.Name,
			"need sync", needSync, "deleted", mpiJob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	// LegacyMPIJobToV1MPIJob handles legacy fields across different versioned mpi-jobs
	// so that job can be reconciled in one implementation.
	if err = LegacyMPIJobToV1MPIJob(mpiJob); err != nil {
		log.Error(err, "handle legacy fields of mpi job failed")
		return result, err
	}

	// Set default properties for tensorflow job.
	r.scheme.Default(mpiJob)

	result, err = r.ctrl.ReconcileJobs(mpiJob, mpiJob.Spec.MPIReplicaSpecs, mpiJob.Status, &mpiJob.Spec.RunPolicy, nil, nil)
	if err != nil {
		log.Error(err, "mpi job reconcile failed")
		return result, err
	}

	return result, err
}

// SetupWithManager setup reconciler to the Manager with default RBAC. The Manager will set fields on the
// Controller and Start it when the Manager is Started.
func (r *MPIJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch owner resource with create event filter.
	if err = c.Watch(&source.Kind{Type: &training.MPIJob{}}, eventhandler.NewEnqueueForObject(r.coordinator), predicate.Funcs{
		CreateFunc: job_controller.OnOwnerCreateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator, r.ctrl.Metrics),
		UpdateFunc: job_controller.OnOwnerUpdateFunc(r.scheme, training.ExtractMetaFieldsFromObject, log, r.coordinator),
		DeleteFunc: job_controller.OnOwnerDeleteFunc(r.ctrl, training.ExtractMetaFieldsFromObject, log),
	}); err != nil {
		return err
	}

	// Watch managed resource with owner and create/delete event filter.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.MPIJob{},
		IsController: true,
	}, predicate.Funcs{
		CreateFunc: r.ctrl.OnPodCreateFunc,
		UpdateFunc: r.ctrl.OnPodUpdateFunc,
		DeleteFunc: r.ctrl.OnPodDeleteFunc,
	}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &training.MPIJob{},
		IsController: true,
	}); err != nil {
		return err
	}
	return nil
}

// ControllerName returns the Controller name
func (r *MPIJobReconciler) ControllerName() string {
	return controllerName
}

// GetAPIGroupVersionKind returns the GroupVersionKind of the API
func (r *MPIJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return training.SchemeGroupVersion.WithKind(training.MPIJobKind)
}

// GetAPIGroupVersion returns the GroupVersion of the API
func (r *MPIJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return training.SchemeGroupVersion
}

// GetGroupNameLabelValue returns the Group Name(value) in the labels of the job
func (r *MPIJobReconciler) GetGroupNameLabelValue() string {
	return training.SchemeGroupVersion.Group
}

// SetClusterSpec generates and sets for the given podTemplateSpec.
func (r *MPIJobReconciler) SetClusterSpec(ctx context.Context, job client.Object, podTemplateSpec *corev1.PodTemplateSpec, rtype, index string) error {
	mpiJob, ok := job.(*training.MPIJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of MPIJob", job)
	}
	workerReplicas := int32(0)
	if workerSpec := mpiJob.Spec.MPIReplicaSpecs[training.MPIReplicaTypeWorker]; workerSpec != nil && workerSpec.Replicas != nil {
		workerReplicas = *workerSpec.Replicas
	}
	if launcherSpec, ok := mpiJob.Spec.MPIReplicaSpecs[training.MPIReplicaTypeLauncher]; ok && launcherSpec != nil {
		// MPIJob ensures job-attached configuration is kept by ConfigMap when launcher
		// neither succeed nor failed.
		log.Info("launcher of MPIJob: ", mpiJob.Namespace+"/"+mpiJob.Name, " generate job config.")
		_, err := r.getOrCreateJobConfig(mpiJob, workerReplicas, launcherRunsWorkload)
		if err != nil {
			return err
		}
	}

	switch rtype {
	case strings.ToLower(string(training.MPIReplicaTypeWorker)):
		r.setupMPIWorker(mpiJob, podTemplateSpec)
	case strings.ToLower(string(training.MPIReplicaTypeLauncher)):
		r.setupMPILauncher(mpiJob, podTemplateSpec)
	}

	return nil
}

// GetDefaultContainerName returns the default container name in pod
func (r *MPIJobReconciler) GetDefaultContainerName() string {
	return training.MPIJobDefaultContainerName
}

// GetDefaultContainerPortName Get the default container port name
func (r *MPIJobReconciler) GetDefaultContainerPortName() string {
	return training.MPIJobDefaultPortName
}

// GetDefaultContainerPortNumber get the default container port number
func (r *MPIJobReconciler) GetDefaultContainerPortNumber() int32 {
	return training.MPIJobDefaultPort
}

// Get replicas reconcile orders so that replica type with higher priority can be created earlier.
func (r *MPIJobReconciler) GetReconcileOrders() []v1.ReplicaType {
	// MPI requires worker created before launcher.
	return []v1.ReplicaType{
		training.MPIReplicaTypeWorker,
		training.MPIReplicaTypeLauncher,
	}
}

// IsMasterRole returns if this replica type with index specified is a master role.
// MasterRole pod will have "job-role=master" set in its label
func (r *MPIJobReconciler) IsMasterRole(replicas map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, index int) bool {
	return false
}

func (r *MPIJobReconciler) setupMPIWorker(mpiJob *training.MPIJob, podTemplateSpec *corev1.PodTemplateSpec) {
	// We need the kubexec.sh script here because Open MPI checks for the path
	// in every rank.
	mode := int32(0555)
	podTemplateSpec.Spec.Volumes = append(podTemplateSpec.Spec.Volumes, corev1.Volume{
		Name: configVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: jobConfigName(mpiJob),
				},
				Items: []corev1.KeyToPath{
					{
						Key:  kubexecScriptName,
						Path: kubexecScriptName,
						Mode: &mode,
					},
				},
			},
		},
	})

	for i := range podTemplateSpec.Spec.Containers {
		container := &podTemplateSpec.Spec.Containers[i]
		if len(container.Command) == 0 && len(container.Args) == 0 {
			container.Command = []string{"sleep"}
			container.Args = []string{"365d"}
		}
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      configVolumeName,
			MountPath: configVolumeMountPath,
		})
	}
}

func (r *MPIJobReconciler) setupMPILauncher(mpiJob *training.MPIJob, podTemplateSpec *corev1.PodTemplateSpec) {
	if len(podTemplateSpec.Spec.Containers) == 0 {
		msg := "launcher pod does not have any containers in its spec"
		log.Error(stderrors.New(msg), msg)
		r.recorder.Event(mpiJob, corev1.EventTypeWarning, "LauncherNotExist", msg)
		return
	}
	// 1. append kubectl delivery init container to delivery kubectl binary file to
	// main containers by shared volume.
	podTemplateSpec.Spec.InitContainers = append(podTemplateSpec.Spec.InitContainers, corev1.Container{
		Name:  "kubectl-delivery",
		Image: kubectlDeliveryImage,
		Env: []corev1.EnvVar{
			{
				Name:  kubectlTargetDirEnv,
				Value: kubectlMountPath,
			},
			{
				Name:  "NAMESPACE",
				Value: mpiJob.Namespace,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(initContainerCpu),
				corev1.ResourceMemory: resource.MustParse(initContainerMem),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      kubectlVolumeName,
				MountPath: kubectlMountPath,
			},
			{
				Name:      configVolumeName,
				MountPath: configVolumeMountPath,
			},
		},
		ImagePullPolicy: corev1.PullIfNotPresent,
	})
	// 2. inject mpi-environments and target delivery volume paths.
	for i := range podTemplateSpec.Spec.Containers {
		setupLauncherMainContainers(mpiJob, &podTemplateSpec.Spec.Containers[i])
	}
	// 3. construct volumes shared across containers and its access mode.
	scriptsMode := int32(0555)
	hostfileMode := int32(0444)
	podTemplateSpec.Spec.Volumes = append(podTemplateSpec.Spec.Volumes, corev1.Volume{
		Name: kubectlVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}, corev1.Volume{
		Name: configVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: jobConfigName(mpiJob)},
				Items: []corev1.KeyToPath{
					{
						Key:  kubexecScriptName,
						Path: kubexecScriptName,
						Mode: &scriptsMode,
					},
					{
						Key:  hostfileName,
						Path: hostfileName,
						Mode: &hostfileMode,
					},
				},
			},
		},
	})
}

func setupLauncherMainContainers(mpiJob *training.MPIJob, container *corev1.Container) {
	// Different MPI frameworks use different environment variables
	// to specify the path of the remote task launcher and hostfile file.
	mpiRshExecPathEnvName := "OMPI_MCA_plm_rsh_agent"
	mpiHostfilePathEnvName := "OMPI_MCA_orte_default_hostfile"
	// If the MPIDistribution is not specificed as the "IntelMPI" or "MPICH",
	// then think that the default "OpenMPI" will be used.
	if mpiJob.Spec.MPIJobLegacySpec != nil && mpiJob.Spec.MPIJobLegacySpec.LegacyV1Alpha2 != nil &&
		mpiJob.Spec.MPIJobLegacySpec.LegacyV1Alpha2.MPIDistribution != nil {
		distribution := *mpiJob.Spec.MPIJobLegacySpec.LegacyV1Alpha2.MPIDistribution

		if distribution == training.MPIDistributionTypeIntelMPI {
			mpiRshExecPathEnvName = "I_MPI_HYDRA_BOOTSTRAP_EXEC"
			mpiHostfilePathEnvName = "I_MPI_HYDRA_HOST_FILE"
		} else if distribution == training.MPIDistributionTypeMPICH {
			mpiRshExecPathEnvName = "HYDRA_LAUNCHER_EXEC"
			mpiHostfilePathEnvName = "HYDRA_HOST_FILE"
		}
	}

	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  mpiRshExecPathEnvName,
			Value: fmt.Sprintf("%s/%s", configVolumeMountPath, kubexecScriptName),
		},
		corev1.EnvVar{
			Name:  mpiHostfilePathEnvName,
			Value: fmt.Sprintf("%s/%s", configVolumeMountPath, hostfileName),
		},
	)

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      kubectlVolumeName,
			MountPath: kubectlMountPath,
		},
		corev1.VolumeMount{
			Name:      configVolumeName,
			MountPath: configVolumeMountPath,
		})
}

func (r *MPIJobReconciler) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	return ""
}
