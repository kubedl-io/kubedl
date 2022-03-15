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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func setDefaults_XDLJobSpec(spec *XDLJobSpec) {
	// Set default clean pod policy to Running.
	if spec.RunPolicy.CleanPodPolicy == nil {
		running := v1.CleanPodPolicyRunning
		spec.RunPolicy.CleanPodPolicy = &running
	}
	// Set MinFinishWorkerPercentage to default value neither MinFinishWorkerNum nor MinFinishWorkerPercentage
	// are set.
	if spec.MinFinishWorkerNum == nil && spec.MinFinishWorkerPercentage == nil {
		spec.MinFinishWorkerPercentage = pointer.Int32Ptr(XDLJobDefaultMinFinishWorkRate)
	}
	// Set MaxFailoverTimes to default value if user not manually set.
	if spec.BackoffLimit == nil {
		spec.BackoffLimit = pointer.Int32Ptr(XDLJobDefaultBackoffLimit)
	}
}

// setDefaults_XDLJobPort sets the default ports for xdl container.
func setDefaults_XDLJobPort(spec *corev1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == XDLJobDefaultContainerName {
			index = i
			break
		}
	}

	hasXDLJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == XDLJobDefaultContainerPortName {
			hasXDLJobPort = true
			break
		}
	}
	if !hasXDLJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, corev1.ContainerPort{
			Name:          XDLJobDefaultContainerPortName,
			ContainerPort: XDLJobDefaultPort,
		})
	}
}

func setTypeNames_XDLJob(xdlJob *XDLJob) {
	setTypeName_XDLJob(xdlJob, XDLReplicaTypeWorker)
	setTypeName_XDLJob(xdlJob, XDLReplicaTypePS)
	setTypeName_XDLJob(xdlJob, XDLReplicaTypeScheduler)
	setTypeName_XDLJob(xdlJob, XDLReplicaTypeExtendRole)
}

// setTypeName_XDLJob sets the name of the replica type from any case to correct case.
// E.g. from ps to PS; from WORKER to Worker.
func setTypeName_XDLJob(xdlJob *XDLJob, typ v1.ReplicaType) {
	for t := range xdlJob.Spec.XDLReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := xdlJob.Spec.XDLReplicaSpecs[t]
			delete(xdlJob.Spec.XDLReplicaSpecs, t)
			xdlJob.Spec.XDLReplicaSpecs[typ] = spec
			return
		}
	}
}

func setDefaults_XDLJobReplicas(spec *v1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = XDLJobDefaultRestartPolicy
	}
}

// SetDefaults_XDLJob sets any unspecified values to defaults.
func SetDefaults_XDLJob(xdlJob *XDLJob) {
	setDefaults_XDLJobSpec(&xdlJob.Spec)
	setTypeNames_XDLJob(xdlJob)

	for _, spec := range xdlJob.Spec.XDLReplicaSpecs {
		enableFallbackToLogsOnErrorTerminationMessagePolicy(&spec.Template.Spec)
		// Set default replica and restart policy.
		setDefaults_XDLJobReplicas(spec)
		// Set default container port for xdl containers
		setDefaults_XDLJobPort(&spec.Template.Spec)
	}

	if xdlJob.Kind == "" {
		xdlJob.Kind = XDLJobKind
	}

	if xdlJob.APIVersion == "" {
		xdlJob.APIVersion = GroupVersion.String()
	}
}
