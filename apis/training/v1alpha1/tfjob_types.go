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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	"github.com/alibaba/kubedl/apis/model/v1alpha1"
	commonv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// TFJobSpec defines the desired state of TFJob
type TFJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	commonv1.RunPolicy `json:",inline"`

	// SuccessPolicy defines the policy to mark the TFJob as succeeded when the job does not contain chief or master
	// role.
	// Value "" means the default policy that the job is succeeded if all workers are succeeded or worker 0 completed,
	// Value "AllWorkers" means the job is succeeded if all workers are succeeded.
	// Default to ""
	// +optional
	SuccessPolicy *commonv1.SuccessPolicy `json:"successPolicy,omitempty"`

	// A map of TFReplicaType (type) to ReplicaSpec (value). Specifies the TF cluster configuration.
	// For example,
	//   {
	//     "PS": ReplicaSpec,
	//     "Worker": ReplicaSpec,
	//   }
	TFReplicaSpecs map[commonv1.ReplicaType]*commonv1.ReplicaSpec `json:"tfReplicaSpecs"`

	// ModelVersion represents the model output by this job run.
	// +optional
	ModelVersion *v1alpha1.ModelVersionSpec `json:"modelVersion,omitempty"`

	// CacheBackend is used to configure the cache engine for job
	// +optional
	CacheBackend *cachev1alpha1.CacheBackendSpec `json:"cacheBackend"`
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

// TFJob is the Schema for the tfjobs API
type TFJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TFJobSpec          `json:"spec,omitempty"`
	Status commonv1.JobStatus `json:"status,omitempty"`
}

// TFReplicaType is the type for TFReplica. Can be one of: "Chief"/"Master" (semantically equivalent),
// "Worker", "PS", or "Evaluator".
type TFReplicaType commonv1.ReplicaType

const (
	// TFReplicaTypePS is the type for parameter servers of distributed TensorFlow.
	TFReplicaTypePS commonv1.ReplicaType = "PS"

	// TFReplicaTypeWorker is the type for workers of distributed TensorFlow.
	// This is also used for non-distributed TensorFlow.
	TFReplicaTypeWorker commonv1.ReplicaType = "Worker"

	// TFReplicaTypeChief is the type for chief worker of distributed TensorFlow.
	// If there is "chief" replica type, it's the "chief worker".
	// Else, worker:0 is the chief worker.
	TFReplicaTypeChief commonv1.ReplicaType = "Chief"

	// TFReplicaTypeMaster is the type for master worker of distributed TensorFlow.
	// This is similar to chief, and kept just for backwards compatibility.
	TFReplicaTypeMaster commonv1.ReplicaType = "Master"

	// TFReplicaTypeEval is the type for evaluation replica in TensorFlow.
	TFReplicaTypeEval commonv1.ReplicaType = "Evaluator"
)

// +kubebuilder:object:root=true

// TFJobList contains a list of TFJob
type TFJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TFJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TFJob{}, &TFJobList{})
}
