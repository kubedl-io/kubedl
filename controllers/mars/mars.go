/*
Copyright 2020 The Alibaba Authors.

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

package controllers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/kubedl/api/marsjob/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/quota"

	corev1 "k8s.io/api/core/v1"
)

const marsConfig = "MARS_TASK_DETAIL"

// MarsConfig is a struct representing the distributed Mars config.
// see https://github.com/mars-project/mars/issues/1458 for details.
type MarsConfig struct {
	Cluster ClusterSpec `json:"cluster"`
	Task    TaskSpec    `json:"task"`
}

type ClusterSpec map[string][]string

type TaskSpec struct {
	Type      string          `json:"type"`
	Index     int             `json:"index"`
	Resources *workerResource `json:"resources,omitempty"`
}

type workerResource struct {
	// Number of computation processes on CPUs.
	CPU int64 `json:"cpu_procs"`
	// Limit of physical memory in megabytes.
	Memory int64 `json:"phy_mem"`
	// Reserved for future use of cuda devices.
	// CUDADevices []string `json:"cuda_devices,omitempty"`
}

func marsConfigInJson(marJob *v1alpha1.MarsJob, rtype, index string) (string, error) {
	i, err := strconv.ParseInt(index, 0, 32)
	if err != nil {
		return "", err
	}

	cluster, err := genClusterSpec(marJob)
	if err != nil {
		return "", err
	}

	task := TaskSpec{
		Type:  rtype,
		Index: int(i),
	}

	// Mars worker will perceive allocated resources and real-time usage inside worker
	// process and report to scheduler.
	if v1.ReplicaType(rtype) == v1alpha1.MarsReplicaTypeWorker {
		resources := quota.ComputePodResourceRequest(&corev1.Pod{
			Spec: *marJob.Spec.MarsReplicaSpecs[v1.ReplicaType(rtype)].Template.Spec.DeepCopy(),
		})
		task.Resources = &workerResource{
			CPU:    resources.Cpu().Value(),
			Memory: resources.Memory().Value() / (1024 * 1024),
		}
	}

	cfg, err := json.Marshal(&MarsConfig{
		Cluster: cluster,
		Task:    task,
	})
	if err != nil {
		return "", err
	}
	return string(cfg), nil
}

func genClusterSpec(marsJob *v1alpha1.MarsJob) (ClusterSpec, error) {
	cs := make(ClusterSpec)

	for rtype, spec := range marsJob.Spec.MarsReplicaSpecs {
		rt := strings.ToLower(string(rtype))
		endpoints := make([]string, 0, *spec.Replicas)

		port, err := job_controller.GetPortFromJob(marsJob.Spec.MarsReplicaSpecs, rtype, v1alpha1.DefaultContainerName, v1alpha1.DefaultPortName)
		if err != nil {
			return nil, err
		}
		for i := int32(0); i < *spec.Replicas; i++ {
			// Headless service assigned a DNS A record for a name of the form "my-svc.my-namespace.svc.cluster.local".
			// And the last part "svc.cluster.local" is called cluster domain
			// which maybe different between kubernetes clusters.
			hostname := job_controller.GenGeneralName(marsJob.Name, rt, fmt.Sprintf("%d", i))
			svcName := hostname + "." + marsJob.Namespace + ".svc"
			endpoint := fmt.Sprintf("%s:%d", svcName, port)
			endpoints = append(endpoints, endpoint)
		}
		cs[rt] = endpoints
	}
	return cs, nil
}
