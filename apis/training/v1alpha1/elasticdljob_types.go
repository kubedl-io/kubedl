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

	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// ElasticDLJobSpec defines the desired state of ElasticDLJob
type ElasticDLJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	common.RunPolicy `json:",inline"`

	// A map of ElasticDLReplicaType (type) to ReplicaSpec (value). Specifies the ElasticDL cluster configuration.
	// For example,
	//   {
	//     "Master": ElasticDLReplicaSpec,
	//   }
	ElasticDLReplicaSpecs map[common.ReplicaType]*common.ReplicaSpec `json:"elasticdlReplicaSpecs"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="TTL-AFTER-FINISHED",type=integer,JSONPath=`.spec.ttlSecondsAfterFinished`
// +kubebuilder:printcolumn:name="Max-Lifetime",type=integer,JSONPath=`.spec.activeDeadlineSeconds`

// ElasticDLJob is the Schema for the elasticdljobs API
type ElasticDLJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticDLJobSpec `json:"spec,omitempty"`
	Status common.JobStatus `json:"status,omitempty"`
}

const (
	// ElasticDLReplicaTypeMaster is the type of Master of distributed ElasticDL
	ElasticDLReplicaTypeMaster common.ReplicaType = "Master"
)

// +kubebuilder:object:root=true

// ElasticDLJobList contains a list of ElasticDLJob
type ElasticDLJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticDLJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticDLJob{}, &ElasticDLJobList{})
}
