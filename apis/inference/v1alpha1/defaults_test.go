/*
Copyright 2022 The Alibaba Authors.

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
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
)

const (
	testImage = "test-image:latest"
)

func expectedElasticBatchJob(cleanPodPolicy common.CleanPodPolicy, portName string, port int32) *ElasticBatchJob {
	ports := []v1.ContainerPort{}

	// port not set
	if portName != "" {
		ports = append(ports,
			v1.ContainerPort{
				Name:          portName,
				ContainerPort: port,
			},
		)
	}

	// port set with custom name
	if portName != ElasticBatchJobDefaultPortName {
		ports = append(ports,
			v1.ContainerPort{
				Name:          ElasticBatchJobDefaultPortName,
				ContainerPort: ElasticBatchJobDefaultPort,
			},
		)
	}

	restartPolicy := common.RestartPolicyExitCode
	defaultSuccessPolicy := common.SuccessPolicyAllWorkers

	return &ElasticBatchJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       ElasticBatchJobKind,
			APIVersion: GroupVersion.String(),
		},
		Spec: ElasticBatchJobSpec{
			SuccessPolicy: &defaultSuccessPolicy,
			RunPolicy:     common.RunPolicy{CleanPodPolicy: &cleanPodPolicy},
			ElasticBatchReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
				ElasticBatchReplicaTypeWorker: {
					Replicas: Int32(1),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:                     ElasticBatchJobDefaultContainerName,
									Image:                    testImage,
									Ports:                    ports,
									TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
								},
							},
						},
					},
					RestartPolicy: restartPolicy,
				},
			},
		},
	}
}

func TestSetTypeNames(t *testing.T) {
	spec := &common.ReplicaSpec{
		RestartPolicy: common.RestartPolicyAlways,
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  ElasticBatchJobDefaultContainerName,
						Image: testImage,
						Ports: []v1.ContainerPort{
							{
								Name:          ElasticBatchJobDefaultPortName,
								ContainerPort: ElasticBatchJobDefaultPort,
							},
						},
					},
				},
			},
		},
	}

	workerUpperCase := common.ReplicaType("WORKER")
	original := &ElasticBatchJob{
		Spec: ElasticBatchJobSpec{
			ElasticBatchReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
				workerUpperCase: spec,
			},
		},
	}

	setTypeNamesToCamelCase(original)
	if _, ok := original.Spec.ElasticBatchReplicaSpecs[workerUpperCase]; ok {
		t.Errorf("Failed to delete key %s", workerUpperCase)
	}
	if _, ok := original.Spec.ElasticBatchReplicaSpecs[ElasticBatchReplicaTypeWorker]; !ok {
		t.Errorf("Failed to set key %s", ElasticBatchReplicaTypeWorker)
	}
}

func TestSetDefaultElasticBatchJob(t *testing.T) {
	customPortName := "customPort"
	var customPort int32 = 1234

	testCases := map[string]struct {
		original *ElasticBatchJob
		expected *ElasticBatchJob
	}{
		"set replicas": {
			original: &ElasticBatchJob{
				Spec: ElasticBatchJobSpec{
					ElasticBatchReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						ElasticBatchReplicaTypeWorker: {
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  ElasticBatchJobDefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												{
													Name:          ElasticBatchJobDefaultPortName,
													ContainerPort: ElasticBatchJobDefaultPort,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedElasticBatchJob(common.CleanPodPolicyNone, ElasticBatchJobDefaultPortName, ElasticBatchJobDefaultPort),
		},
		"set replicas with default restartpolicy": {
			original: &ElasticBatchJob{
				Spec: ElasticBatchJobSpec{
					ElasticBatchReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						ElasticBatchReplicaTypeWorker: {
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  ElasticBatchJobDefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												{
													Name:          ElasticBatchJobDefaultPortName,
													ContainerPort: ElasticBatchJobDefaultPort,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedElasticBatchJob(common.CleanPodPolicyNone, ElasticBatchJobDefaultPortName, ElasticBatchJobDefaultPort),
		},
		"set replicas with default port": {
			original: &ElasticBatchJob{
				Spec: ElasticBatchJobSpec{
					ElasticBatchReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						ElasticBatchReplicaTypeWorker: {
							Replicas: Int32(1),
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  ElasticBatchJobDefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												{Name: ElasticBatchJobDefaultPortName, ContainerPort: ElasticBatchJobDefaultPort},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedElasticBatchJob(common.CleanPodPolicyNone, "", 0),
		},
		"set replicas adding default port": {
			original: &ElasticBatchJob{
				Spec: ElasticBatchJobSpec{
					ElasticBatchReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						ElasticBatchReplicaTypeWorker: {
							Replicas: Int32(1),
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  ElasticBatchJobDefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												{Name: customPortName, ContainerPort: customPort},
												{Name: ElasticBatchJobDefaultPortName, ContainerPort: ElasticBatchJobDefaultPort},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedElasticBatchJob(common.CleanPodPolicyNone, customPortName, customPort),
		},
		"set custom cleanpod policy": {
			original: &ElasticBatchJob{
				Spec: ElasticBatchJobSpec{
					RunPolicy: common.RunPolicy{CleanPodPolicy: cleanPodPolicyPointer(common.CleanPodPolicyAll)},
					ElasticBatchReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						ElasticBatchReplicaTypeWorker: {
							Replicas: Int32(1),
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  ElasticBatchJobDefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												{
													Name:          customPortName,
													ContainerPort: customPort,
												},
												{
													Name:          ElasticBatchJobDefaultPortName,
													ContainerPort: ElasticBatchJobDefaultPort,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedElasticBatchJob(common.CleanPodPolicyAll, customPortName, customPort),
		},
	}

	for name, tc := range testCases {
		SetDefaults_ElasticBatchJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, util.Pformat(tc.expected), util.Pformat(tc.original))
		}
	}
}

func cleanPodPolicyPointer(cleanPodPolicy common.CleanPodPolicy) *common.CleanPodPolicy {
	c := cleanPodPolicy
	return &c
}
