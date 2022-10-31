/*Copyright 2022 The Alibaba Authors.

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
	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ElasticBatchJobSpec defines the desired state of ElasticBatchJob
type ElasticBatchJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	common.RunPolicy `json:",inline"`

	// SuccessPolicy defines the policy to mark the ElasticBatchJob as succeeded when the job contains master role.
	// Value "" means the default policy that the job is succeeded if all workers are succeeded or master completed,
	// Value "AllWorkers" means the job is succeeded if all workers *AND* master are succeeded.
	// Default to ""
	// +optional
	SuccessPolicy *common.SuccessPolicy `json:"successPolicy,omitempty"`

	// A map of ElasticBatchReplicaType (type) to ReplicaSpec (value). Specifies the ElasticBatchJob cluster configuration.
	// For example,
	//   {
	//      "AIMaster": ReplicaSpec,
	//      "Worker": ReplicaSpec,
	//   }
	ElasticBatchReplicaSpecs map[common.ReplicaType]*common.ReplicaSpec `json:"elasticBatchReplicaSpecs"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Model-Version",type=string,JSONPath=`.status.modelVersionName`
// +kubebuilder:printcolumn:name="Cache-Backend",type=string,JSONPath=`.status.cacheBackendName`
// +kubebuilder:printcolumn:name="Max-Lifetime",type=integer,JSONPath=`.spec.activeDeadlineSeconds`
// +kubebuilder:printcolumn:name="TTL-AFTER-FINISHED",type=integer,JSONPath=`.spec.ttlSecondsAfterFinished`

// Represents a ElasticBatchJob resource.
type ElasticBatchJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticBatchJobSpec `json:"spec,omitempty"`
	Status common.JobStatus    `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:resource:scope=Namespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ElasticBatchJobList contains a list of ElasticBatchJob

type ElasticBatchJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticBatchJob `json:"items"`
}

const (
	// ElasticBatchReplicaTypeAIMaster is the type of AIMaster of distributed ElasticBatchJob
	ElasticBatchReplicaTypeAIMaster common.ReplicaType = common.JobReplicaTypeAIMaster

	// ElasticBatchReplicaTypeWorker is the type for workers of distributed ElasticBatchJob.
	ElasticBatchReplicaTypeWorker common.ReplicaType = "Worker"
)

func init() {
	SchemeBuilder.Register(&ElasticBatchJob{}, &ElasticBatchJobList{})
}
