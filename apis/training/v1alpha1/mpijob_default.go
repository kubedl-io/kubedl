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

	"github.com/alibaba/kubedl/pkg/features"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// setDefaultPort sets the default ports for mpi container.
func setDefaults_MPIJobPort(spec *corev1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == MPIJobDefaultContainerName {
			index = i
			break
		}
	}

	hasMPIJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == MPIJobDefaultPortName {
			hasMPIJobPort = true
			break
		}
	}
	if !hasMPIJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, corev1.ContainerPort{
			Name:          MPIJobDefaultPortName,
			ContainerPort: MPIJobDefaultPort,
		})
	}
}

// setDefaults_MPIJobLauncherReplica sets the default value to launcher.
func setDefaults_MPIJobLauncherReplica(spec *v1.ReplicaSpec) {
	if spec != nil {
		if spec.RestartPolicy == "" {
			spec.RestartPolicy = MPIJobDefaultRestartPolicy
		}
		setDefaults_MPIJobPort(&spec.Template.Spec)
		enableFallbackToLogsOnErrorTerminationMessagePolicy(&spec.Template.Spec)
	}
}

// setDefaults_MPIJobWorkerReplica sets the default value to worker.
func setDefaults_MPIJobWorkerReplica(spec *v1.ReplicaSpec) {
	if spec != nil {
		if spec.RestartPolicy == "" {
			spec.RestartPolicy = MPIJobDefaultRestartPolicy
		}
		setDefaults_MPIJobPort(&spec.Template.Spec)
		enableFallbackToLogsOnErrorTerminationMessagePolicy(&spec.Template.Spec)
	}
}

func setDefaultMPIDAGConditions(job *MPIJob) {
	// DAG scheduling flow for mpi job:
	//
	// Worker
	//  |---> Launcher
	if job.Spec.MPIReplicaSpecs[MPIReplicaTypeWorker] != nil &&
		job.Spec.MPIReplicaSpecs[MPIReplicaTypeLauncher] != nil {
		job.Spec.MPIReplicaSpecs[MPIReplicaTypeLauncher].DependOn = []v1.DAGCondition{
			{Upstream: MPIReplicaTypeWorker, OnPhase: corev1.PodRunning},
		}
	}
}

func SetDefaults_MPIJob(mpiJob *MPIJob) {
	// Set default cleanpod policy to None.
	if mpiJob.Spec.MPIJobLegacySpec != nil && mpiJob.Spec.MPIJobLegacySpec.RunPolicy == nil {
		mpiJob.Spec.MPIJobLegacySpec.RunPolicy = &v1.RunPolicy{}
	}
	if mpiJob.Spec.MPIJobLegacySpec != nil && mpiJob.Spec.MPIJobLegacySpec.CleanPodPolicy == nil {
		policy := MPIJobDefaultCleanPodPolicy
		mpiJob.Spec.CleanPodPolicy = &policy
	}

	// set default to Launcher
	if mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeLauncher] != nil {
		setDefaults_MPIJobLauncherReplica(mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeLauncher])
	}

	// set default to Worker
	if mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeWorker] != nil {
		setDefaults_MPIJobWorkerReplica(mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeWorker])
	}

	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		setDefaultMPIDAGConditions(mpiJob)
	}

	if mpiJob.Kind == "" {
		mpiJob.Kind = MPIJobKind
	}

	if mpiJob.APIVersion == "" {
		mpiJob.APIVersion = GroupVersion.String()
	}
}
