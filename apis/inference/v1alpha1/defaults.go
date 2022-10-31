// Copyright 2022 The Kubeflow Authors
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
	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

// setDefaultPort sets the default ports for elasticbatch container.
func setDefaultPort(spec *corev1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == ElasticBatchJobDefaultContainerName {
			index = i
			break
		}
	}

	hasElasticBatchJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == ElasticBatchJobDefaultPortName {
			hasElasticBatchJobPort = true
			break
		}
	}
	if !hasElasticBatchJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, corev1.ContainerPort{
			Name:          ElasticBatchJobDefaultPortName,
			ContainerPort: ElasticBatchJobDefaultPort,
		})
	}
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func setDefaultAIMasterReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil || *spec.Replicas != 1 {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = ElasticBatchJobDefaultAIMasterRestartPolicy
	}
}

func setDefaultWorkerReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = ElasticBatchJobDefaultWorkerRestartPolicy
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setTypeNamesToCamelCase(job *ElasticBatchJob) {
	setTypeNameToCamelCase(job, ElasticBatchReplicaTypeWorker)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
func setTypeNameToCamelCase(job *ElasticBatchJob, typ common.ReplicaType) {
	for t := range job.Spec.ElasticBatchReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := job.Spec.ElasticBatchReplicaSpecs[t]
			delete(job.Spec.ElasticBatchReplicaSpecs, t)
			job.Spec.ElasticBatchReplicaSpecs[typ] = spec
			return
		}
	}
}

func setDefaultDAGConditions(job *ElasticBatchJob) {
	// DAG scheduling flow for ElasticBatch job
	//
	// AIMaster
	//  |--> Worker

	if job.Spec.ElasticBatchReplicaSpecs[common.JobReplicaTypeAIMaster] != nil {
		for rtype := range job.Spec.ElasticBatchReplicaSpecs {
			if rtype == common.JobReplicaTypeAIMaster {
				continue
			}
			job.Spec.ElasticBatchReplicaSpecs[rtype].DependOn = append(job.Spec.ElasticBatchReplicaSpecs[rtype].DependOn, common.DAGCondition{
				Upstream: common.JobReplicaTypeAIMaster,
				OnPhase:  corev1.PodRunning,
			})
		}
	}
}

func setDefaultAIMasterEnv(job *ElasticBatchJob, spec *corev1.PodSpec) {
	jobName := job.ObjectMeta.Name
	jobNamespace := job.ObjectMeta.Namespace
	for i := range spec.Containers {
		if spec.Containers[i].Name == ElasticBatchJobDefaultContainerName {
			if len(spec.Containers[i].Env) == 0 {
				spec.Containers[i].Env = make([]corev1.EnvVar, 0)
			}
			spec.Containers[i].Env = append(spec.Containers[i].Env, corev1.EnvVar{
				Name:  "JOB_NAME",
				Value: jobName,
			})
			spec.Containers[i].Env = append(spec.Containers[i].Env, corev1.EnvVar{
				Name:  "NAMESPACE",
				Value: jobNamespace,
			})
			spec.Containers[i].Env = append(spec.Containers[i].Env, corev1.EnvVar{
				Name:  "JOB_TYPE",
				Value: "ELASTICBATCH",
			})
			break
		}
	}
}

func EnableFallbackToLogsOnErrorTerminationMessagePolicy(podSpec *corev1.PodSpec) {
	for ci := range podSpec.Containers {
		c := &podSpec.Containers[ci]
		if c.TerminationMessagePolicy == "" {
			c.TerminationMessagePolicy = corev1.TerminationMessageFallbackToLogsOnError
		}
	}
}

// SetDefaults_ElasticBatchJob sets any unspecified values to defaults.
func SetDefaults_ElasticBatchJob(job *ElasticBatchJob) {
	if job.Kind == "" {
		job.Kind = ElasticBatchJobKind
	}
	if job.APIVersion == "" {
		job.APIVersion = GroupVersion.String()
	}
	// Set default cleanpod policy to None.
	if job.Spec.CleanPodPolicy == nil {
		policy := common.CleanPodPolicyNone
		job.Spec.CleanPodPolicy = &policy
	}

	// Set default success policy to "AllWorkers".
	if job.Spec.SuccessPolicy == nil {
		defaultPolicy := common.SuccessPolicyAllWorkers
		job.Spec.SuccessPolicy = &defaultPolicy
	}

	// Update the key of ElasticBatchReplicaSpecs to camel case.
	setTypeNamesToCamelCase(job)

	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		setDefaultDAGConditions(job)
	}

	for rType, spec := range job.Spec.ElasticBatchReplicaSpecs {
		EnableFallbackToLogsOnErrorTerminationMessagePolicy(&spec.Template.Spec)
		// Set default replicas and restart policy.
		if rType == ElasticBatchReplicaTypeWorker {
			setDefaultWorkerReplicas(spec)
		} else if rType == ElasticBatchReplicaTypeAIMaster {
			setDefaultAIMasterReplicas(spec)
			setDefaultPort(&spec.Template.Spec)
			setDefaultAIMasterEnv(job, &spec.Template.Spec)
		}
	}
}
