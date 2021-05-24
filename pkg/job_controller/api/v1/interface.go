package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ControllerInterface defines the Interface to be implemented by custom operators. e.g. tf-operator needs to implement this interface
type ControllerInterface interface {
	// Returns the Controller name
	ControllerName() string

	// Returns the GroupVersionKind of the API
	GetAPIGroupVersionKind() schema.GroupVersionKind

	// Returns the GroupVersion of the API
	GetAPIGroupVersion() schema.GroupVersion

	// Returns the Group Name(value) in the labels of the job
	GetGroupNameLabelValue() string

	// Returns the Job from Informer Cache
	GetJobFromInformerCache(namespace, name string) (v1.Object, error)

	// Returns the Job from API server
	GetJobFromAPIClient(namespace, name string) (v1.Object, error)

	// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
	// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
	GetPodsForJob(job interface{}) ([]*corev1.Pod, error)

	// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
	// i.e. all services created by the job will come with label "job-name" = <this_job_name>
	GetServicesForJob(job interface{}) ([]*corev1.Service, error)

	// GetNodeForModelOutput returns the nodeName where the model is output, in case of local storage.
	// If model is output in remote storage, this will return "Any".
	GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string)

	// DeleteJob deletes the job
	DeleteJob(job interface{}) error

	// UpdateJobStatus updates the job status and job conditions
	UpdateJobStatus(job interface{}, replicas map[ReplicaType]*ReplicaSpec, jobStatus *JobStatus, restart bool) error

	// UpdateJobStatusInApiServer updates the job status in API server
	UpdateJobStatusInApiServer(job interface{}, jobStatus *JobStatus) error

	// SetClusterSpec sets the cluster spec for the pod
	SetClusterSpec(ctx context.Context, job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error

	// Returns the default container name in pod
	GetDefaultContainerName() string

	// Get the default container port name
	GetDefaultContainerPortName() string

	// Get the default container port number
	GetDefaultContainerPortNumber() int32

	// Get replicas reconcile orders so that replica type with higher priority can be created earlier.
	GetReconcileOrders() []ReplicaType

	// Returns if this replica type with index specified is a master role.
	// MasterRole pod will have "job-role=master" set in its label
	IsMasterRole(replicas map[ReplicaType]*ReplicaSpec, rtype ReplicaType, index int) bool
}
