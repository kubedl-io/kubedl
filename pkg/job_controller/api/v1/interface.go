package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControllerInterface defines the Interface to be implemented by custom operators. e.g. tf-operator needs to implement this interface
type ControllerInterface interface {
	// ControllerName Returns the Controller name
	ControllerName() string

	// GetAPIGroupVersionKind Returns the GroupVersionKind of the API
	GetAPIGroupVersionKind() schema.GroupVersionKind

	// GetAPIGroupVersion Returns the GroupVersion of the API
	GetAPIGroupVersion() schema.GroupVersion

	// GetGroupNameLabelValue Returns the Group Name(value) in the labels of the job
	GetGroupNameLabelValue() string

	// GetJobFromInformerCache Returns the Job from Informer Cache
	GetJobFromInformerCache(namespace, name string) (client.Object, error)

	// GetJobFromAPIClient Returns the Job from API server
	GetJobFromAPIClient(namespace, name string) (client.Object, error)

	// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
	// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
	GetPodsForJob(job client.Object) ([]*corev1.Pod, error)

	// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
	// i.e. all services created by the job will come with label "job-name" = <this_job_name>
	GetServicesForJob(job client.Object) ([]*corev1.Service, error)

	// GetNodeForModelOutput returns the nodeName where the model is output, in case of local storage.
	// If model is output in remote storage, this will return "Any".
	GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string)

	// UpdateJobStatus updates the job status and job conditions
	UpdateJobStatus(job client.Object, replicas map[ReplicaType]*ReplicaSpec, jobStatus *JobStatus, restart bool) error

	// UpdateJobStatusInApiServer updates the job status in API server
	UpdateJobStatusInApiServer(job client.Object, jobStatus *JobStatus) error

	// SetClusterSpec sets the cluster spec for the pod
	SetClusterSpec(ctx context.Context, job client.Object, podTemplate *corev1.PodTemplateSpec, rtype, index string) error

	// GetDefaultContainerName Returns the default container name in pod
	GetDefaultContainerName() string

	// GetDefaultContainerPortName Get the default container port name
	GetDefaultContainerPortName() string

	// GetDefaultContainerPortNumber Get the default container port number
	GetDefaultContainerPortNumber() int32

	// GetReconcileOrders Get replicas reconcile orders so that replica type with higher priority can be created earlier.
	GetReconcileOrders() []ReplicaType

	// IsMasterRole Returns if this replica type with index specified is a master role.
	// MasterRole pod will have "job-role=master" set in its label
	IsMasterRole(replicas map[ReplicaType]*ReplicaSpec, rtype ReplicaType, index int) bool

	ElasticScaling
}

// ElasticScaling defines the interface to be implemented by custom workload elastic behaviors.
type ElasticScaling interface {
	// EnableElasticScaling indicates workload enables elastic scaling or not.
	EnableElasticScaling(job client.Object, runPolicy *RunPolicy) bool

	// ScaleOut defines how to scale out a job instance(i.e. scale workers from n to 2*n), usually
	// the scaling progress is incremental and the implementation guarantees idempotence.
	ScaleOut(job client.Object, replicas map[ReplicaType]*ReplicaSpec, activePods []*corev1.Pod, activeServices []*corev1.Service) error

	// ScaleIn defines how to scale in a job instance(i.e. scale workers from 2*n to n), usually
	// the scaling progress is incremental and the implementation guarantees idempotence.
	ScaleIn(job client.Object, replicas map[ReplicaType]*ReplicaSpec, activePods []*corev1.Pod, activeServices []*corev1.Service) error

	// CheckpointIfNecessary triggers job checkpoints when it is necessary, e.g. workers are going to be
	// preempted after a grace termination period.
	CheckpointIfNecessary(job client.Object, activePods []*corev1.Pod) (completed bool, err error)
}
