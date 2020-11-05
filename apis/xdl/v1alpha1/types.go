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
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// XDLJobSpec defines the desired state of XDLJob
type XDLJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	v1.RunPolicy `json:",inline"`

	// XDLReplicaSpecs is map of ReplicaType and ReplicaSpec
	// specifies the XDL replicas to run.
	// For example,
	//   {
	//     "PS": ReplicaSpec,
	//     "Worker": ReplicaSpec,
	//   }
	XDLReplicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec `json:"xdlReplicaSpecs"`

	// MinFinishWorkerNum specifies the minimum number of successfully finished
	// workers such that the job is treated as successful. Not specifying this
	// value means all worker should be successfully finished.
	MinFinishWorkerNum *int32 `json:"minFinishWorkNum,omitempty"`

	// MinFinishWorkPercentage specifies the minimum percentage of all workers
	// that should be finished successfully such that the job is treated as successful.
	// MinFinishWorkPercentage takes precedence over  MinFinishWorkerNum if both are
	// specified.
	MinFinishWorkerPercentage *int32 `json:"minFinishWorkRate,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=TypeMeta
// +resource:path=xdljob
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Finished-TTL",type=integer,JSONPath=`.spec.ttlSecondsAfterFinished`
// +kubebuilder:printcolumn:name="Max-Lifetime",type=integer,JSONPath=`.spec.activeDeadlineSeconds`

// XDLJob is the Schema for the xdljobs API
// +k8s:openapi-gen=true
type XDLJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XDLJobSpec   `json:"spec,omitempty"`
	Status v1.JobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XDLJobList contains a list of XDLJob
type XDLJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []XDLJob `json:"items"`
}

const (
	// XDLReplicaTypePS is the type for parameter servers of distributed XDL.
	XDLReplicaTypePS v1.ReplicaType = "PS"

	// XDLReplicaTypeWorker is the type for workers of distributed XDL.
	// This is also used for non-distributed XDL.
	XDLReplicaTypeWorker v1.ReplicaType = "Worker"

	// XDLReplicaTypeScheduler is the type for code auto-generation scheduler of
	// distributed XDL.
	XDLReplicaTypeScheduler v1.ReplicaType = "Scheduler"

	// XDLReplicaTypeExtendRole is the extended replica type of distributed XDL.
	// ExtendRole may participate in computing and be seen as another kind of
	// worker.
	XDLReplicaTypeExtendRole v1.ReplicaType = "ExtendRole"
)

func init() {
	SchemeBuilder.Register(&XDLJob{}, &XDLJobList{})
}
