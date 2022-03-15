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
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
)

const (
	testTFJobImage = "test-image:latest"
)

func expectedTFJob(cleanPodPolicy common.CleanPodPolicy, restartPolicy common.RestartPolicy, portName string, port int32) *TFJob {
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
	if portName != TFJobDefaultPortName {
		ports = append(ports,
			v1.ContainerPort{
				Name:          TFJobDefaultPortName,
				ContainerPort: TFJobDefaultPort,
			},
		)
	}

	defaultSuccessPolicy := common.SuccessPolicyDefault

	return &TFJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       TFJobKind,
			APIVersion: GroupVersion.String(),
		},
		Spec: TFJobSpec{
			SuccessPolicy: &defaultSuccessPolicy,
			RunPolicy:     common.RunPolicy{CleanPodPolicy: &cleanPodPolicy},
			TFReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
				TFReplicaTypeWorker: &common.ReplicaSpec{
					Replicas:      pointer.Int32Ptr(1),
					RestartPolicy: restartPolicy,
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:                     TFJobDefaultContainerName,
									Image:                    testTFJobImage,
									Ports:                    ports,
									TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestSetTypeNames_TFJob(t *testing.T) {
	spec := &common.ReplicaSpec{
		RestartPolicy: common.RestartPolicyAlways,
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  TFJobDefaultContainerName,
						Image: testTFJobImage,
						Ports: []v1.ContainerPort{
							{
								Name:          TFJobDefaultPortName,
								ContainerPort: TFJobDefaultPort,
							},
						},
					},
				},
			},
		},
	}

	workerUpperCase := common.ReplicaType("WORKER")
	original := &TFJob{
		Spec: TFJobSpec{
			TFReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
				workerUpperCase: spec,
			},
		},
	}

	setTypeNames_TFJob(original)
	if _, ok := original.Spec.TFReplicaSpecs[workerUpperCase]; ok {
		t.Errorf("Failed to delete key %s", workerUpperCase)
	}
	if _, ok := original.Spec.TFReplicaSpecs[TFReplicaTypeWorker]; !ok {
		t.Errorf("Failed to set key %s", TFReplicaTypeWorker)
	}
}

func TestSetDefaults_TFJob(t *testing.T) {
	customPortName := "customPort"
	var customPort int32 = 1234
	customRestartPolicy := common.RestartPolicyAlways

	testCases := map[string]struct {
		original *TFJob
		expected *TFJob
	}{
		"set replicas": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						TFReplicaTypeWorker: &common.ReplicaSpec{
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  TFJobDefaultContainerName,
											Image: testTFJobImage,
											Ports: []v1.ContainerPort{
												{
													Name:          TFJobDefaultPortName,
													ContainerPort: TFJobDefaultPort,
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
			expected: expectedTFJob(common.CleanPodPolicyRunning, customRestartPolicy, TFJobDefaultPortName, TFJobDefaultPort),
		},
		"set replicas with default restartpolicy": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						TFReplicaTypeWorker: &common.ReplicaSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  TFJobDefaultContainerName,
											Image: testTFJobImage,
											Ports: []v1.ContainerPort{
												{
													Name:          TFJobDefaultPortName,
													ContainerPort: TFJobDefaultPort,
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
			expected: expectedTFJob(common.CleanPodPolicyRunning, TFJobDefaultRestartPolicy, TFJobDefaultPortName, TFJobDefaultPort),
		},
		"set replicas with default port": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						TFReplicaTypeWorker: &common.ReplicaSpec{
							Replicas:      pointer.Int32Ptr(1),
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  TFJobDefaultContainerName,
											Image: testTFJobImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedTFJob(common.CleanPodPolicyRunning, customRestartPolicy, "", 0),
		},
		"set replicas adding default port": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						TFReplicaTypeWorker: &common.ReplicaSpec{
							Replicas:      pointer.Int32Ptr(1),
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  TFJobDefaultContainerName,
											Image: testTFJobImage,
											Ports: []v1.ContainerPort{
												{
													Name:          customPortName,
													ContainerPort: customPort,
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
			expected: expectedTFJob(common.CleanPodPolicyRunning, customRestartPolicy, customPortName, customPort),
		},
		"set custom cleanpod policy": {
			original: &TFJob{
				Spec: TFJobSpec{
					RunPolicy: common.RunPolicy{CleanPodPolicy: cleanPodPolicyPointer(common.CleanPodPolicyAll)},
					TFReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
						TFReplicaTypeWorker: &common.ReplicaSpec{
							Replicas:      pointer.Int32Ptr(1),
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  TFJobDefaultContainerName,
											Image: testTFJobImage,
											Ports: []v1.ContainerPort{
												{
													Name:          customPortName,
													ContainerPort: customPort,
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
			expected: expectedTFJob(common.CleanPodPolicyAll, customRestartPolicy, customPortName, customPort),
		},
	}

	for name, tc := range testCases {
		SetDefaults_TFJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, util.Pformat(tc.expected), util.Pformat(tc.original))
		}
	}
}
