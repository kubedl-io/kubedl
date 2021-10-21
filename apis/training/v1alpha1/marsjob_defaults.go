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

	"github.com/alibaba/kubedl/pkg/features"
	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

// setDefaults_MarsJobPort sets the default ports for pytorch container.
func setDefaults_MarsJobPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == MarsJobDefaultContainerName {
			index = i
			break
		}
	}

	hasMarsJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == MarsJobDefaultPortName {
			hasMarsJobPort = true
			break
		}
	}
	if !hasMarsJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          MarsJobDefaultPortName,
			ContainerPort: MarsJobDefaultPort,
		})
	}
}

func setDefaults_MarsJobSchedulerReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = MarsJobDefaultSchedulerRestartPolicy
	}
}

func setDefaults_MarsJobWorkerReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = MarsJobDefaultWorkerRestartPolicy
	}
}

func setDefaults_MarsJobWebServiceReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = MarsJobDefaultWebServiceRestartPolicy
	}
}

func setDefaults_MarsJobWorkerMemTuningPolicy(job *MarsJob) {
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

func setDefaultMarsDAGConditions(job *MarsJob) {
	// DAG scheduling flow for mars job:
	//
	// Scheduler
	//  |---> Web
	//  |---> Worker
	if job.Spec.MarsReplicaSpecs[MarsReplicaTypeWorker] != nil &&
		job.Spec.MarsReplicaSpecs[MarsReplicaTypeScheduler] != nil {
		job.Spec.MarsReplicaSpecs[MarsReplicaTypeWorker].DependOn = []common.DAGCondition{
			{Upstream: MarsReplicaTypeScheduler, OnPhase: v1.PodRunning},
		}
	}
	if job.Spec.MarsReplicaSpecs[MarsReplicaTypeWebService] != nil &&
		job.Spec.MarsReplicaSpecs[MarsReplicaTypeScheduler] != nil {
		job.Spec.MarsReplicaSpecs[MarsReplicaTypeWebService].DependOn = []common.DAGCondition{
			{Upstream: MarsReplicaTypeScheduler, OnPhase: v1.PodRunning},
		}
	}
}

// setTypeNames_MarsJob sets the name of all replica types from any case to correct case.
func setTypeNames_MarsJob(job *MarsJob) {
	setTypeName_MarsJob(job, MarsReplicaTypeScheduler)
	setTypeName_MarsJob(job, MarsReplicaTypeWorker)
	setTypeName_MarsJob(job, MarsReplicaTypeWebService)
}

// setTypeName_MarsJob sets the name of the replica type from any case to correct case.
func setTypeName_MarsJob(job *MarsJob, typ common.ReplicaType) {
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
	setTypeNames_MarsJob(job)
	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		setDefaultMarsDAGConditions(job)
	}

	for rType, spec := range job.Spec.MarsReplicaSpecs {
		setDefaults_MarsJobPort(&spec.Template.Spec)
		enableFallbackToLogsOnErrorTerminationMessagePolicy(&spec.Template.Spec)
		switch rType {
		case MarsReplicaTypeWebService:
			setDefaults_MarsJobWebServiceReplicas(spec)
		case MarsReplicaTypeWorker:
			setDefaults_MarsJobWorkerReplicas(spec)
			setDefaults_MarsJobWorkerMemTuningPolicy(job)
		case MarsReplicaTypeScheduler:
			setDefaults_MarsJobSchedulerReplicas(spec)
		}
	}

	if job.Kind == "" {
		job.Kind = MarsJobKind
	}

	if job.APIVersion == "" {
		job.APIVersion = GroupVersion.String()
	}
}
