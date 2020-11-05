/*
Copyright 2020 The Alibaba Authors.

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
	"strings"

	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// setDefaultPort sets the default ports for pytorch container.
func setDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == DefaultContainerName {
			index = i
			break
		}
	}

	hasMarsJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultPortName {
			hasMarsJobPort = true
			break
		}
	}
	if !hasMarsJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          DefaultPortName,
			ContainerPort: DefaultPort,
		})
	}
}

func setDefaultSchedulerReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = DefaultSchedulerRestartPolicy
	}
}

func setDefaultWorkerReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = DefaultWorkerRestartPolicy
	}
}

func setDefaultWebServiceReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = DefaultWebServiceRestartPolicy
	}
}

func setDefaultWorkerMemTuningPolicy(job *MarsJob) {
	if job.Spec.WorkerMemoryTuningPolicy == nil {
		job.Spec.WorkerMemoryTuningPolicy = &MarsWorkerMemoryTuningPolicy{}
	}
	if job.Spec.WorkerMemoryTuningPolicy.PlasmaStore == nil {
		job.Spec.WorkerMemoryTuningPolicy.PlasmaStore = pointer.StringPtr("/dev/shm")
	}
	if job.Spec.WorkerMemoryTuningPolicy.LockFreeFileIO == nil {
		job.Spec.WorkerMemoryTuningPolicy.LockFreeFileIO = pointer.BoolPtr(true)
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setTypeNamesToCamelCase(job *MarsJob) {
	setTypeNameToCamelCase(job, MarsReplicaTypeScheduler)
	setTypeNameToCamelCase(job, MarsReplicaTypeWorker)
	setTypeNameToCamelCase(job, MarsReplicaTypeWebService)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
func setTypeNameToCamelCase(job *MarsJob, typ common.ReplicaType) {
	for t := range job.Spec.MarsReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := job.Spec.MarsReplicaSpecs[t]
			delete(job.Spec.MarsReplicaSpecs, t)
			job.Spec.MarsReplicaSpecs[typ] = spec
			return
		}
	}
}

// SetDefaults_MarsJob sets any unspecified values to defaults.
func SetDefaults_MarsJob(job *MarsJob) {
	// Set default cleanpod policy to None.
	if job.Spec.CleanPodPolicy == nil {
		policy := common.CleanPodPolicyNone
		job.Spec.CleanPodPolicy = &policy
	}

	// Update the key of PyTorchReplicaSpecs to camel case.
	setTypeNamesToCamelCase(job)

	for rType, spec := range job.Spec.MarsReplicaSpecs {
		setDefaultPort(&spec.Template.Spec)
		switch rType {
		case MarsReplicaTypeWebService:
			setDefaultWebServiceReplicas(spec)
		case MarsReplicaTypeWorker:
			setDefaultWorkerReplicas(spec)
			setDefaultWorkerMemTuningPolicy(job)
		case MarsReplicaTypeScheduler:
			setDefaultSchedulerReplicas(spec)
		}
	}
}
