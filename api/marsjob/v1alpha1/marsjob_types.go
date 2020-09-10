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

package v1alpha1

import (
	commonv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MarsJobSpec defines the desired state of MarsJob
type MarsJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	commonv1.RunPolicy `json:",inline"`

	// WorkerMemoryTuningPolicy provides multiple memory tuning policies to mars worker
	// spec, such as cache size, cold data paths...
	WorkerMemoryTuningPolicy *MarsWorkerMemoryTuningPolicy `json:"workerMemoryTuningPolicy,omitempty"`

	// WebHost is the domain address of webservice that expose to external users.
	WebHost *string `json:"webHost,omitempty"`

	// MarsReplicaSpecs is a map of MarsReplicaType(key) to ReplicaSpec(value),
	// specifying replicas and template of each type.
	MarsReplicaSpecs map[commonv1.ReplicaType]*commonv1.ReplicaSpec `json:"marsReplicaSpecs"`
}

// MarsWorkerMemoryTuningPolicy defines memory tuning policies that will be applied
// to workers.
type MarsWorkerMemoryTuningPolicy struct {
	// PlasmaStore specify the socket path of plasma store that handles shared memory
	// between all worker processes.
	PlasmaStore *string `json:"plasmaStore,omitempty"`

	// LockFreeFileIO indicates whether spill dirs are dedicated or not.
	LockFreeFileIO *bool `json:"lockFreeFileIO,omitempty"`

	// SpillDirs specify multiple directory paths, when size of in-memory objects is
	// about to reach the limitation, mars workers will swap cold data out to spill dirs
	// and persist in ephemeral-storage.
	SpillDirs []string `json:"spillDirs,omitempty"`

	// WorkerCachePercentage specify the percentage of total available memory size can
	// be used as cache, it will be overridden by workerCacheSize if it is been set.
	WorkerCachePercentage *int32 `json:"workerCachePercentage,omitempty"`

	// WorkerCacheSize specify the exact cache quantity can be used.
	WorkerCacheSize *resource.Quantity `json:"workerCacheSize,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Finished-TTL",type=integer,JSONPath=`.spec.ttlSecondsAfterFinished`
// +kubebuilder:printcolumn:name="Max-Lifetime",type=integer,JSONPath=`.spec.activeDeadlineSeconds`

// MarsJob represents a mars job instance nad
type MarsJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MarsJobSpec        `json:"spec,omitempty"`
	Status commonv1.JobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=TypeMeta
// +resource:path=pytorchjob
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:object:root=true

// MarsJobList contains a list of MarsJob
type MarsJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MarsJob `json:"items"`
}

const (
	// MarsReplicaTypeScheduler is the type for scheduler role in MarsJob, schedule
	// graph-based workflow including 'operand' and 'chunk' to workers.
	MarsReplicaTypeScheduler commonv1.ReplicaType = "Scheduler"

	// MarsReplicaTypeWorker is the type for accepting the scheduled operand from
	// scheduler do the real execution, it will pull data(chunk) from mounted storage
	// or other workers, and notify its execution result to scheduler by callback.
	MarsReplicaTypeWorker commonv1.ReplicaType = "Worker"

	// MarsReplicaTypeWebService is the type for web-service instance, accepting
	// requests from end user and forwarding the whole tensor-graph to scheduler.
	// WebService provides end users with a dashboard so that they can track job
	// status and submit tensor-graph tasks interactively.
	MarsReplicaTypeWebService commonv1.ReplicaType = "WebService"
)

func init() {
	SchemeBuilder.Register(&MarsJob{}, &MarsJobList{})
}
