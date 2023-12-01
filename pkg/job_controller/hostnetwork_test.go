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

package job_controller

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"

	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
)

func TestAppendOrOverrideContainerPort(t *testing.T) {
	testCases := []struct {
		name          string
		spec          *v1.PodTemplateSpec
		containerName string
		portName      string
		port          int32
		expected      *v1.PodTemplateSpec
	}{
		{
			name: "single container with default names",
			spec: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: trainingv1alpha1.TFJobDefaultContainerName,
					portName:      trainingv1alpha1.TFJobDefaultPortName,
					port:          trainingv1alpha1.TFJobDefaultPort,
				},
			}),
			containerName: trainingv1alpha1.TFJobDefaultContainerName,
			portName:      trainingv1alpha1.TFJobDefaultPortName,
			port:          1111,
			expected: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: trainingv1alpha1.TFJobDefaultContainerName,
					portName:      trainingv1alpha1.TFJobDefaultPortName,
					port:          1111,
				},
			}),
		},
		{
			name: "single container with custom names",
			spec: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: "main",
					portName:      trainingv1alpha1.TFJobDefaultPortName,
					port:          trainingv1alpha1.TFJobDefaultPort,
				},
			}),
			containerName: trainingv1alpha1.TFJobDefaultContainerName,
			portName:      trainingv1alpha1.TFJobDefaultPortName,
			port:          1111,
			expected: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: "main",
					portName:      trainingv1alpha1.TFJobDefaultPortName,
					port:          1111,
				},
			}),
		},
		{
			name: "multiple container with default names",
			spec: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: trainingv1alpha1.TFJobDefaultContainerName,
					portName:      trainingv1alpha1.TFJobDefaultPortName,
					port:          trainingv1alpha1.TFJobDefaultPort,
				},
				{
					containerName: "other",
					portName:      "other",
					port:          1111,
				},
			}),
			containerName: trainingv1alpha1.TFJobDefaultContainerName,
			portName:      trainingv1alpha1.TFJobDefaultPortName,
			port:          1234,
			expected: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: trainingv1alpha1.TFJobDefaultContainerName,
					portName:      trainingv1alpha1.TFJobDefaultPortName,
					port:          1234,
				},
				{
					containerName: "other",
					portName:      "other",
					port:          1111,
				},
			}),
		},
		{
			name: "multiple container with custom names",
			spec: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: "custom",
				},
			}),
			containerName: trainingv1alpha1.TFJobDefaultContainerName,
			portName:      trainingv1alpha1.TFJobDefaultPortName,
			port:          1234,
			expected: newPodTemplateSpec([]simpleContainerPort{
				{
					containerName: "custom",
					portName:      trainingv1alpha1.TFJobDefaultPortName,
					port:          1234,
				},
			}),
		},
	}

	for _, testCase := range testCases {
		setupContainerHostNetworkPort(testCase.spec, testCase.containerName, testCase.portName, testCase.port)
		if !reflect.DeepEqual(testCase.spec, testCase.expected) {
			t.Errorf("case: %s, expected: %s, got: %s", testCase.name, testCase.expected, testCase.spec)
		}
	}
}

type simpleContainerPort struct {
	containerName string
	portName      string
	port          int32
}

func newPodTemplateSpec(containerPorts []simpleContainerPort) *v1.PodTemplateSpec {
	p := &v1.PodTemplateSpec{Spec: v1.PodSpec{Containers: []v1.Container{}}}
	for _, cp := range containerPorts {
		c := v1.Container{Name: cp.containerName}
		if cp.portName != "" {
			c.Ports = []v1.ContainerPort{
				{
					Name:          cp.portName,
					ContainerPort: cp.port,
					HostPort:      cp.port,
				},
			}
		}

		p.Spec.Containers = append(p.Spec.Containers, c)
	}
	return p
}
