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
	corev1 "k8s.io/api/core/v1"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MPIJobSpec defines the desired state of MPIJob
type MPIJobSpec struct {
	// Specifies the number of slots per worker used in hostfile.
	// Defaults to 1.
	// +optional
	SlotsPerWorker *int32 `json:"slotsPerWorker,omitempty"`

	// `MPIReplicaSpecs` contains maps from `MPIReplicaType` to `ReplicaSpec` that
	// specify the MPI replicas to run.
	MPIReplicaSpecs map[apiv1.ReplicaType]*apiv1.ReplicaSpec `json:"mpiReplicaSpecs"`

	// MainContainer specifies name of the main container which
	// executes the MPI code.
	MainContainer string `json:"mainContainer,omitempty"`

	// `RunPolicy` encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	RunPolicy apiv1.RunPolicy `json:"runPolicy,omitempty"`

	// LegacySpec reserves the deprecated fields for backward compatibility.
	*MPIJobLegacySpec `json:",inline"`
}

// MPIJobLegacySpec is a collection of legacy fields that were used in v1alpha1/v1alpha2 but
// deprecated in v1 version, we reserve legacy fields for backward compatibility.
type MPIJobLegacySpec struct {
	// RunPolicy is inline embedded in MPIJobSpec in both v1alpha1.
	*apiv1.RunPolicy `json:",inline"`

	// LegacyV1Alpha1 is legacy fields in v1alpha1 api definition.
	*LegacyV1Alpha1 `json:",inline"`

	// LegacyV1Alpha2 is legacy fields in v1alpha2 api definition.
	*LegacyV1Alpha2 `json:",inline"`
}

type LegacyV1Alpha1 struct {
	// Deprecated. Specifies the desired number of DeprecatedGPUs the MPIJob should run on.
	// Mutually exclusive with the `Replicas` field.
	// Note that this is deprecated in favor of `ProcessingUnits` field.
	// +optional
	DeprecatedGPUs *int32 `json:"gpus,omitempty"`

	// The maximum number of GPUs available per node.
	// Note that this will be ignored if the GPU resources are explicitly
	// specified in the MPIJob pod spec.
	// This is deprecated in favor of `ProcessingUnitsPerNode` field.
	GPUsPerNode *int32 `json:"gpusPerNode,omitempty"`

	// Specifies the desired number of processing units the MPIJob should run on.
	// Mutually exclusive with the `Replicas` field.
	// +optional
	ProcessingUnits *int32 `json:"processingUnits,omitempty"`

	// The maximum number of processing units available per node.
	// Note that this will be ignored if the processing resources are explicitly
	// specified in the MPIJob pod spec.
	// +optional
	ProcessingUnitsPerNode *int32 `json:"processingUnitsPerNode,omitempty"`

	// The processing resource type, e.g. 'nvidia.com/gpu' or 'cpu'.
	// Defaults to 'nvidia.com/gpu'
	// +optional
	ProcessingResourceType string `json:"processingResourceType,omitempty"`

	// Run the launcher on the master.
	// Defaults to false.
	// +optional
	LauncherOnMaster bool `json:"launcherOnMaster,omitempty"`

	// Specifies the desired number of replicas the MPIJob should run on.
	// The `PodSpec` should specify the number of processing units.
	// Mutually exclusive with the `GPUs` or `ProcessingUnits` fields.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Describes the pod that will be created when executing an MPIJob.
	Template corev1.PodTemplateSpec `json:"template,omitempty"`
}

type LegacyV1Alpha2 struct {
	// MPIDistribution specifies name of the MPI framwork which is used
	// Defaults to "OpenMPI"
	// Options includes "OpenMPI", "IntelMPI" and "MPICH"
	MPIDistribution *MPIDistributionType `json:"mpiDistribution,omitempty"`
}

const (
	// MPIReplicaTypeLauncher is the type for launcher replica.
	MPIReplicaTypeLauncher apiv1.ReplicaType = "Launcher"

	// MPIReplicaTypeWorker is the type for worker replicas.
	MPIReplicaTypeWorker apiv1.ReplicaType = "Worker"
)

// MPIDistributionType is the type for MPIDistribution.
type MPIDistributionType string

const (
	// MPIDistributionTypeOpenMPI is the type for Open MPI.
	MPIDistributionTypeOpenMPI MPIDistributionType = "OpenMPI"

	// MPIDistributionTypeIntelMPI is the type for Intel MPI.
	MPIDistributionTypeIntelMPI MPIDistributionType = "IntelMPI"

	// MPIDistributionTypeMPICH is the type for MPICh.
	MPIDistributionTypeMPICH MPIDistributionType = "MPICH"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="TTL-AFTER-FINISHED",type=integer,JSONPath=`.spec.ttlSecondsAfterFinished`
// +kubebuilder:printcolumn:name="Max-Lifetime",type=integer,JSONPath=`.spec.activeDeadlineSeconds`

// MPIJob is the Schema for the mpijobs API
type MPIJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MPIJobSpec      `json:"spec,omitempty"`
	Status apiv1.JobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MPIJobList contains a list of MPIJob
type MPIJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MPIJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MPIJob{}, &MPIJobList{})
}
