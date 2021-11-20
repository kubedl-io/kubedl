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
	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// PyTorchJobSpec defines the desired state of PyTorchJob
type PyTorchJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	common.RunPolicy `json:",inline"`

	// A map of PyTorchReplicaType (type) to ReplicaSpec (value). Specifies the PyTorch cluster configuration.
	// For example,
	//   {
	//     "Master": PyTorchReplicaSpec,
	//     "Worker": PyTorchReplicaSpec,
	//   }
	PyTorchReplicaSpecs map[common.ReplicaType]*common.ReplicaSpec `json:"pytorchReplicaSpecs"`

	// ModelVersion represents the model output by this job run.
	// +optional
	ModelVersion *v1alpha1.ModelVersionSpec `json:"modelVersion,omitempty"`

	// CacheBackend is used to configure the cache engine for job
	// +optional
	CacheBackend *cachev1alpha1.CacheBackendSpec `json:"cacheBackend"`
}

// PyTorchJobStatus defines the observed state of PyTorchJob
type PyTorchJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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

// PyTorchJob is the Schema for the pytorchjobs API
type PyTorchJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PyTorchJobSpec   `json:"spec,omitempty"`
	Status common.JobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PyTorchJobList contains a list of PyTorchJob
type PyTorchJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PyTorchJob `json:"items"`
}

const (
	// PyTorchReplicaTypeMaster is the type of Master of distributed PyTorch
	PyTorchReplicaTypeMaster common.ReplicaType = "Master"

	// PyTorchReplicaTypeWorker is the type for workers of distributed PyTorch.
	PyTorchReplicaTypeWorker common.ReplicaType = "Worker"
)

func init() {
	SchemeBuilder.Register(&PyTorchJob{}, &PyTorchJobList{})
}
