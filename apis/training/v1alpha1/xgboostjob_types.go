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

	commonv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// XGBoostJobSpec defines the desired state of XGBoostJob
type XGBoostJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	RunPolicy commonv1.RunPolicy `json:",inline"`

	// XGBoostReplicaSpecs is map of ReplicaType and ReplicaSpec
	// specifies the XGBoost replicas to run.
	// For example,
	//   {
	//     "PS": ReplicaSpec,
	//     "Worker": ReplicaSpec,
	//   }
	XGBReplicaSpecs map[commonv1.ReplicaType]*commonv1.ReplicaSpec `json:"xgbReplicaSpecs"`
}

// XGBoostJobStatus defines the observed state of XGBoostJob
type XGBoostJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	commonv1.JobStatus `json:",inline"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="TTL-AFTER-FINISHED",type=integer,JSONPath=`.spec.ttlSecondsAfterFinished`
// +kubebuilder:printcolumn:name="Max-Lifetime",type=integer,JSONPath=`.spec.activeDeadlineSeconds`

// XGBoostJob is the Schema for the xgboostjobs API
type XGBoostJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XGBoostJobSpec   `json:"spec,omitempty"`
	Status XGBoostJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// XGBoostJobList contains a list of XGBoostJob
type XGBoostJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []XGBoostJob `json:"items"`
}

const (
	// XGBoostReplicaTypeMaster is the type of Master of distributed PyTorch
	XGBoostReplicaTypeMaster commonv1.ReplicaType = "Master"

	// XGBoostReplicaTypeWorker is the type for workers of distributed PyTorch.
	XGBoostReplicaTypeWorker commonv1.ReplicaType = "Worker"
)

func init() {
	SchemeBuilder.Register(&XGBoostJob{}, &XGBoostJobList{})
}
