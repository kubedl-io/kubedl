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

	"github.com/alibaba/kubedl/pkg/features"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func setDefaults_XGBoostJobSpec(spec *XGBoostJobSpec) {
	// Set default clean pod policy to Running.
	if spec.RunPolicy.CleanPodPolicy == nil {
		policy := XGBoostJobDefaultCleanPodPolicy
		spec.RunPolicy.CleanPodPolicy = &policy
	}
	if spec.RunPolicy.TTLSecondsAfterFinished == nil {
		ttl := XGBoostJobDefaultTTLseconds
		spec.RunPolicy.TTLSecondsAfterFinished = &ttl
	}
}

// setDefaults_XGBoostJobPort sets the default ports for xgboost container.
func setDefaults_XGBoostJobPort(spec *corev1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == XGBoostJobDefaultContainerName {
			index = i
			break
		}
	}

	hasJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == XGBoostJobDefaultContainerPortName {
			hasJobPort = true
			break
		}
	}
	if !hasJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, corev1.ContainerPort{
			Name:          XGBoostJobDefaultContainerPortName,
			ContainerPort: XGBoostJobDefaultPort,
		})
	}
}

func setTypeNames_XGBoostJob(xgbJob *XGBoostJob) {
	setTypeName_XGBoostJob(xgbJob, XGBoostReplicaTypeMaster)
	setTypeName_XGBoostJob(xgbJob, XGBoostReplicaTypeWorker)
}

// setTypeName_XGBoostJob sets the name of the replica type from any case to correct case.
// E.g. from ps to PS; from WORKER to Worker.
func setTypeName_XGBoostJob(xgbJob *XGBoostJob, typ v1.ReplicaType) {
	for t := range xgbJob.Spec.XGBReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := xgbJob.Spec.XGBReplicaSpecs[t]
			delete(xgbJob.Spec.XGBReplicaSpecs, t)
			xgbJob.Spec.XGBReplicaSpecs[typ] = spec
			return
		}
	}
}

func setDefaults_XGBoostJobReplicas(spec *v1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = pointer.Int32Ptr(1)
	}
}

func setDefaultXGBoostDAGConditions(job *XGBoostJob) {
	// DAG scheduling flow for xgboost job:
	//
	//  Master
	//  |--> Worker
	if job.Spec.XGBReplicaSpecs[XGBoostReplicaTypeMaster] != nil &&
		job.Spec.XGBReplicaSpecs[XGBoostReplicaTypeWorker] != nil {
		job.Spec.XGBReplicaSpecs[XGBoostReplicaTypeWorker].DependOn = []v1.DAGCondition{
			{Upstream: XGBoostReplicaTypeMaster, OnPhase: corev1.PodRunning},
		}
	}
}

// SetDefaults_XGBoostJob sets any unspecified values to defaults.
func SetDefaults_XGBoostJob(xgbJob *XGBoostJob) {
	setDefaults_XGBoostJobSpec(&xgbJob.Spec)
	setTypeNames_XGBoostJob(xgbJob)

	for _, spec := range xgbJob.Spec.XGBReplicaSpecs {
		enableFallbackToLogsOnErrorTerminationMessagePolicy(&spec.Template.Spec)
		// Set default replica and restart policy.
		setDefaults_XGBoostJobReplicas(spec)
		// Set default container port for xgboost containers
		setDefaults_XGBoostJobPort(&spec.Template.Spec)
	}

	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		setDefaultXGBoostDAGConditions(xgbJob)
	}

	if xgbJob.Kind == "" {
		xgbJob.Kind = XGBoostJobKind
	}

	if xgbJob.APIVersion == "" {
		xgbJob.APIVersion = GroupVersion.String()
	}
}
