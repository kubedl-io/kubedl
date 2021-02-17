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

package v1

import (
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// setDefaultPort sets the default ports for mpi container.
func setDefaultPort(spec *corev1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == DefaultContainerName {
			index = i
			break
		}
	}

	hasMPIJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultPortName {
			hasMPIJobPort = true
			break
		}
	}
	if !hasMPIJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, corev1.ContainerPort{
			Name:          DefaultPortName,
			ContainerPort: DefaultPort,
		})
	}
}

// setDefaultsTypeLauncher sets the default value to launcher.
func setDefaultsTypeLauncher(spec *v1.ReplicaSpec) {
	if spec != nil {
		if spec.RestartPolicy == "" {
			spec.RestartPolicy = DefaultRestartPolicy
		}
		setDefaultPort(&spec.Template.Spec)
	}
}

// setDefaultsTypeWorker sets the default value to worker.
func setDefaultsTypeWorker(spec *v1.ReplicaSpec) {
	if spec != nil {
		if spec.RestartPolicy == "" {
			spec.RestartPolicy = DefaultRestartPolicy
		}
		setDefaultPort(&spec.Template.Spec)
	}
}

func SetDefaults_MPIJob(mpiJob *MPIJob) {
	// Set default cleanpod policy to None.
	if mpiJob.Spec.CleanPodPolicy == nil {
		policy := DefaultCleanPodPolicy
		mpiJob.Spec.CleanPodPolicy = &policy
	}

	// set default to Launcher
	setDefaultsTypeLauncher(mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeLauncher])

	// set default to Worker
	setDefaultsTypeWorker(mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeWorker])
}
