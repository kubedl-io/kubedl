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
	"strings"

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

func setDefaultXGBJobSpec(spec *XGBoostJobSpec) {
	// Set default clean pod policy to Running.
	if spec.RunPolicy.CleanPodPolicy == nil {
		policy := DefaultCleanPodPolicy
		spec.RunPolicy.CleanPodPolicy = &policy
	}
	if spec.RunPolicy.TTLSecondsAfterFinished == nil {
		ttl := DefaultTTLseconds
		spec.RunPolicy.TTLSecondsAfterFinished = &ttl
	}
}

// setDefaultPort sets the default ports for xgboost container.
func setDefaultPort(spec *corev1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == DefaultContainerName {
			index = i
			break
		}
	}

	hasJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultContainerPortName {
			hasJobPort = true
			break
		}
	}
	if !hasJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, corev1.ContainerPort{
			Name:          DefaultContainerPortName,
			ContainerPort: DefaultPort,
		})
	}
}

func setDefaultReplicas(spec *v1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
}

func setTypeNamesToCamelCase(xgbJob *XGBoostJob) {
	setTypeNameToCamelCase(xgbJob, XGBoostReplicaTypeMaster)
	setTypeNameToCamelCase(xgbJob, XGBoostReplicaTypeWorker)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from ps to PS; from WORKER to Worker.
func setTypeNameToCamelCase(xgbJob *XGBoostJob, typ v1.ReplicaType) {
	for t := range xgbJob.Spec.XGBReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := xgbJob.Spec.XGBReplicaSpecs[t]
			delete(xgbJob.Spec.XGBReplicaSpecs, t)
			xgbJob.Spec.XGBReplicaSpecs[typ] = spec
			return
		}
	}
}

// SetDefaults_XGBJob sets any unspecified values to defaults.
func SetDefaults_XGBJob(xgbJob *XGBoostJob) {
	setDefaultXGBJobSpec(&xgbJob.Spec)
	setTypeNamesToCamelCase(xgbJob)

	for _, spec := range xgbJob.Spec.XGBReplicaSpecs {
		// Set default replica and restart policy.
		setDefaultReplicas(spec)
		// Set default container port for xgboost containers
		setDefaultPort(&spec.Template.Spec)
	}
}
