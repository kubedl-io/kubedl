// Copyright 2018 The Kubeflow Authors
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

package v1alpha1

import (
	"strings"

	"github.com/alibaba/kubedl/pkg/features"

	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// setDefaults_PyTorchJobPort sets the default ports for pytorch container.
func setDefaults_PyTorchJobPort(spec *v1.PodSpec) {
	index := -1
	for i, container := range spec.Containers {
		if container.Name == PyTorchJobDefaultContainerName {
			index = i
			break
		}
	}

	if index < 0 {
		return
	}

	hasPyTorchJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == PyTorchJobDefaultPortName {
			hasPyTorchJobPort = true
			break
		}
	}
	if !hasPyTorchJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          PyTorchJobDefaultPortName,
			ContainerPort: PyTorchJobDefaultPort,
		})
	}
}
func setDefaults_PyTorchJobMasterReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = PyTorchJobDefaultMasterRestartPolicy
	}
}

func setDefaults_PyTorchJobWorkerReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = PyTorchJobDefaultWorkerRestartPolicy
	}
}

// setTypeNames_PyTorchJob sets the name of all replica types from any case to correct case.
func setTypeNames_PyTorchJob(job *PyTorchJob) {
	setTypeName_PyTorchJob(job, PyTorchReplicaTypeMaster)
	setTypeName_PyTorchJob(job, PyTorchReplicaTypeWorker)
}

// setTypeName_PyTorchJob sets the name of the replica type from any case to correct case.
func setTypeName_PyTorchJob(job *PyTorchJob, typ common.ReplicaType) {
	for t := range job.Spec.PyTorchReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := job.Spec.PyTorchReplicaSpecs[t]
			delete(job.Spec.PyTorchReplicaSpecs, t)
			job.Spec.PyTorchReplicaSpecs[typ] = spec
			return
		}
	}
}

func setDefaultPyTorchDAGConditions(job *PyTorchJob) {
	// DAG scheduling flow for pytorch job:
	//
	//  Master
	//  |--> Worker

	if job.Spec.PyTorchReplicaSpecs[common.JobReplicaTypeAIMaster] != nil &&
		job.Spec.PyTorchReplicaSpecs[PyTorchReplicaTypeMaster] != nil {
		job.Spec.PyTorchReplicaSpecs[PyTorchReplicaTypeMaster].DependOn = []common.DAGCondition{
			{Upstream: common.JobReplicaTypeAIMaster, OnPhase: v1.PodRunning},
		}
	}

	if job.Spec.PyTorchReplicaSpecs[PyTorchReplicaTypeWorker] != nil &&
		job.Spec.PyTorchReplicaSpecs[PyTorchReplicaTypeMaster] != nil {
		job.Spec.PyTorchReplicaSpecs[PyTorchReplicaTypeWorker].DependOn = []common.DAGCondition{
			{Upstream: PyTorchReplicaTypeMaster, OnPhase: v1.PodRunning},
		}
	}
}

// SetDefaults_PyTorchJob sets any unspecified values to defaults.
func SetDefaults_PyTorchJob(job *PyTorchJob) {
	// Set default cleanpod policy to None.
	if job.Spec.CleanPodPolicy == nil {
		policy := common.CleanPodPolicyNone
		job.Spec.CleanPodPolicy = &policy
	}

	// Update the key of PyTorchReplicaSpecs to camel case.
	setTypeNames_PyTorchJob(job)

	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		setDefaultPyTorchDAGConditions(job)
	}

	for rType, spec := range job.Spec.PyTorchReplicaSpecs {
		// Set default replicas and restart policy.
		if rType == PyTorchReplicaTypeWorker {
			setDefaults_PyTorchJobWorkerReplicas(spec)
		}
		if rType == PyTorchReplicaTypeMaster {
			setDefaults_PyTorchJobMasterReplicas(spec)
			// Set default port to pytorch container of Master.
			setDefaults_PyTorchJobPort(&spec.Template.Spec)
		}
		enableFallbackToLogsOnErrorTerminationMessagePolicy(&spec.Template.Spec)
	}

	if job.Kind == "" {
		job.Kind = PyTorchJobKind
	}

	if job.APIVersion == "" {
		job.APIVersion = GroupVersion.String()
	}
}
