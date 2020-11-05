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
	"reflect"
	"testing"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

const (
	testImage = "test-image:latest"
)

func expectedXDLJob(
	cleanPodPolicy v1.CleanPodPolicy,
	restartPolicy v1.RestartPolicy,
	portName string,
	port int32,
	minFinishWorkNum,
	minFinishWorkRate,
	backoffLimit *int32) *XDLJob {
	ports := make([]corev1.ContainerPort, 0)

	// port not set
	if portName != "" {
		ports = append(ports,
			corev1.ContainerPort{
				Name:          portName,
				ContainerPort: port,
			},
		)
	}

	// port set with custom name
	if portName != DefaultContainerPortName {
		ports = append(ports,
			corev1.ContainerPort{
				Name:          DefaultContainerPortName,
				ContainerPort: DefaultPort,
			},
		)
	}

	return &XDLJob{
		Spec: XDLJobSpec{
			RunPolicy: v1.RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
				BackoffLimit:   backoffLimit,
			},
			XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
				XDLReplicaTypeWorker: &v1.ReplicaSpec{
					Replicas:      Int32(1),
					RestartPolicy: restartPolicy,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  DefaultContainerName,
									Image: testImage,
									Ports: ports,
								},
							},
						},
					},
				},
			},
			MinFinishWorkerNum:        minFinishWorkNum,
			MinFinishWorkerPercentage: minFinishWorkRate,
		},
	}
}

func TestSetTypeNames(t *testing.T) {
	spec := &v1.ReplicaSpec{
		RestartPolicy: v1.RestartPolicyAlways,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  DefaultContainerName,
						Image: testImage,
						Ports: []corev1.ContainerPort{
							{
								Name:          DefaultContainerPortName,
								ContainerPort: DefaultPort,
							},
						},
					},
				},
			},
		},
	}

	workerUpperCase := v1.ReplicaType("WORKER")
	original := &XDLJob{
		Spec: XDLJobSpec{
			RunPolicy: v1.RunPolicy{BackoffLimit: Int32(DefaultBackoffLimit)},
			XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
				workerUpperCase: spec,
			},
			MinFinishWorkerNum:        Int32(DefaultMinFinishWorkNum),
			MinFinishWorkerPercentage: Int32(DefaultMinFinishWorkRate),
		},
	}

	setTypeNamesToCamelCase(original)
	if _, ok := original.Spec.XDLReplicaSpecs[workerUpperCase]; ok {
		t.Errorf("Failed to delete key %s", workerUpperCase)
	}
	if _, ok := original.Spec.XDLReplicaSpecs[XDLReplicaTypeWorker]; !ok {
		t.Errorf("Failed to set key %s", XDLReplicaTypeWorker)
	}
}

func TestSetDefaultTFJob(t *testing.T) {
	customPortName := "customPort"
	var customPort int32 = 1234
	customRestartPolicy := v1.RestartPolicyAlways
	customMinFinishWorkNum := Int32(10)
	customMinFinishWorkRate := Int32(100)
	customBackoffLimit := Int32(100)

	testCases := map[string]struct {
		original *XDLJob
		expected *XDLJob
	}{
		"set replicas": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							RestartPolicy: customRestartPolicy,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          DefaultContainerPortName,
													ContainerPort: DefaultPort,
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
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				customRestartPolicy,
				DefaultContainerPortName,
				DefaultPort,
				nil,
				Int32(DefaultMinFinishWorkRate),
				Int32(DefaultBackoffLimit),
			),
		},
		"set replicas with default restart policy": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          DefaultContainerPortName,
													ContainerPort: DefaultPort,
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
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				DefaultRestartPolicy,
				DefaultContainerPortName,
				DefaultPort,
				nil,
				Int32(DefaultMinFinishWorkRate),
				Int32(DefaultBackoffLimit),
			),
		},
		"set replicas with default port": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Replicas:      Int32(1),
							RestartPolicy: customRestartPolicy,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				customRestartPolicy,
				"",
				0,
				nil,
				Int32(DefaultMinFinishWorkRate),
				Int32(DefaultBackoffLimit),
			),
		},
		"set replicas adding default port": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Replicas:      Int32(1),
							RestartPolicy: customRestartPolicy,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
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
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				customRestartPolicy,
				customPortName,
				customPort,
				nil,
				Int32(DefaultMinFinishWorkRate),
				Int32(DefaultBackoffLimit),
			),
		},
		"set custom clean pod policy": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					RunPolicy: v1.RunPolicy{CleanPodPolicy: cleanPodPolicyPointer(v1.CleanPodPolicyAll)},
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Replicas:      Int32(1),
							RestartPolicy: customRestartPolicy,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
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
			expected: expectedXDLJob(v1.CleanPodPolicyAll,
				customRestartPolicy,
				customPortName,
				customPort,
				nil,
				Int32(DefaultMinFinishWorkRate),
				Int32(DefaultBackoffLimit),
			),
		},
		"set default min finish attributes": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          DefaultContainerPortName,
													ContainerPort: DefaultPort,
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
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				DefaultRestartPolicy,
				DefaultContainerPortName,
				DefaultPort,
				nil,
				Int32(DefaultMinFinishWorkRate),
				Int32(DefaultBackoffLimit),
			),
		},
		"set add min finish work num": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          DefaultContainerPortName,
													ContainerPort: DefaultPort,
												},
											},
										},
									},
								},
							},
						},
					},
					MinFinishWorkerNum: customMinFinishWorkNum,
				},
			},
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				DefaultRestartPolicy,
				DefaultContainerPortName,
				DefaultPort,
				customMinFinishWorkNum,
				nil,
				Int32(DefaultBackoffLimit),
			),
		},
		"set add min finish work percentage": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          DefaultContainerPortName,
													ContainerPort: DefaultPort,
												},
											},
										},
									},
								},
							},
						},
					},
					MinFinishWorkerPercentage: customMinFinishWorkRate,
				},
			},
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				DefaultRestartPolicy,
				DefaultContainerPortName,
				DefaultPort,
				nil,
				customMinFinishWorkRate,
				Int32(DefaultBackoffLimit),
			),
		},
		"set add backoff limit": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          DefaultContainerPortName,
													ContainerPort: DefaultPort,
												},
											},
										},
									},
								},
							},
						},
					},
					RunPolicy: v1.RunPolicy{BackoffLimit: customBackoffLimit},
				},
			},
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				DefaultRestartPolicy,
				DefaultContainerPortName,
				DefaultPort,
				nil,
				Int32(DefaultMinFinishWorkRate),
				customBackoffLimit,
			),
		},
		"set add bakcoff limit per type": {
			original: &XDLJob{
				Spec: XDLJobSpec{
					XDLReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
						XDLReplicaTypeWorker: &v1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          DefaultContainerPortName,
													ContainerPort: DefaultPort,
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
			expected: expectedXDLJob(
				v1.CleanPodPolicyRunning,
				DefaultRestartPolicy,
				DefaultContainerPortName,
				DefaultPort,
				nil,
				Int32(DefaultMinFinishWorkRate),
				Int32(DefaultBackoffLimit),
			),
		},
	}

	for name, tc := range testCases {
		SetDefaults_XDLJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, util.Pformat(tc.expected), util.Pformat(tc.original))
		}
	}
}

func cleanPodPolicyPointer(cleanPodPolicy v1.CleanPodPolicy) *v1.CleanPodPolicy {
	return &cleanPodPolicy
}
