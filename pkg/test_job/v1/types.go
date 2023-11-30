// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=testjob

// A generic job used for unit tests.
type TestJob struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the TestJob.
	Spec TestJobSpec `json:"spec,omitempty"`

	// Most recently observed status of the TestJob.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	Status apiv1.JobStatus `json:"status,omitempty"`
}

var _ runtime.Object = &TestJob{}

// TestJobSpec is a desired state description of the TestJob.
type TestJobSpec struct {
	RunPolicy        *apiv1.RunPolicy                         `json:"runPolicy,omitempty"`
	TestReplicaSpecs map[apiv1.ReplicaType]*apiv1.ReplicaSpec `json:"testReplicaSpecs"`
}

const (
	TestReplicaTypeWorker apiv1.ReplicaType = "Worker"
	TestReplicaTypeMaster apiv1.ReplicaType = "Master"
	Testmode              apiv1.NetworkMode = "host"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=testjobs

// TestJobList is a list of TestJobs.
type TestJobList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of TestJobs.
	Items []TestJob `json:"items"`
}
