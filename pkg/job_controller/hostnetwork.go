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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
)

func EnableHostNetwork(job metav1.Object) bool {
	return job.GetAnnotations()[v1.AnnotationNetworkMode] == string(v1.HostNetworkMode)
}

func GetHostNetworkPortFromContext(ctx context.Context, rtype, index string) (int32, bool) {
	ports := ctx.Value(v1.ContextHostNetworkPorts).(map[string]int32)
	port, ok := ports[fmt.Sprintf("%s-%s", rtype, index)]
	return port, ok
}

func storeHostNetworkPortToContext(ctx context.Context, rtype, index string, port int32) {
	ports := ctx.Value(v1.ContextHostNetworkPorts).(map[string]int32)
	ports[fmt.Sprintf("%s-%s", rtype, index)] = port
}

func setupContainerHostNetworkPort(spec *corev1.PodTemplateSpec, defaultContainerName, defaultPortName string, port int32) {
	if len(spec.Spec.Containers) == 0 {
		return
	}
	ci := 0
	for index := 1; index < len(spec.Spec.Containers); index++ {
		if spec.Spec.Containers[index].Name == defaultContainerName {
			ci = index
			break
		}
	}
	pi := -1
	for index, port := range spec.Spec.Containers[ci].Ports {
		if port.Name == defaultPortName {
			pi = index
			break
		}
	}
	// Override existed container port with a new value, if specified
	// port not exists then append a new one.
	if pi < 0 {
		spec.Spec.Containers[ci].Ports = append(spec.Spec.Containers[ci].Ports, corev1.ContainerPort{
			Name:          defaultPortName,
			HostPort:      port,
			ContainerPort: port,
		})
	} else {
		spec.Spec.Containers[ci].Ports[pi].ContainerPort = port
		spec.Spec.Containers[ci].Ports[pi].HostPort = port
	}
}

func getContainerHostNetworkPort(pod *corev1.Pod, defaultContainerName, defaultPortName string) int32 {
	if len(pod.Spec.Containers) == 0 {
		util.LoggerForPod(pod, "").Warningf("pod %s/%s containers is empty", pod.Namespace, pod.Name)
		return -1
	}
	if !pod.Spec.HostNetwork {
		util.LoggerForPod(pod, "").Warningf("pod %s/%s enabled hostnetwork but disabled in its spec", pod.Namespace, pod.Name)
	}

	ci := 0
	for index := 1; index < len(pod.Spec.Containers); index++ {
		if pod.Spec.Containers[index].Name == defaultContainerName {
			ci = index
			break
		}
	}
	pi := 0
	for index := 1; index < len(pod.Spec.Containers[ci].Ports); index++ {
		if pod.Spec.Containers[ci].Ports[pi].Name == defaultPortName {
			pi = index
			break
		}
	}
	return pod.Spec.Containers[ci].Ports[pi].ContainerPort
}
