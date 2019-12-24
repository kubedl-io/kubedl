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

package xdljob

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/kubedl/api/xdl/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
)

// XDLConfig is a struct representing the distributed XDL config.
// This struct is turned into an environment variable XDL_CONFIG
// which is used by XDL processes to configure themselves.
type XDLConfig struct {
	// Cluster represents a XDL ClusterSpec.
	Cluster ClusterSpec `json:"cluster"`
	Task    TaskSpec    `json:"task"`
}

// ClusterSpec represents a cluster XDL specification.
// It is a map from job names to network addresses.
type ClusterSpec map[string][]string

// TaskSpec is the specification for a task (PS or worker) of the XDLJob.
type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// genXDLConfigJSON will generate the environment variable json formatted XDL_CONFIG
// {
//     "cluster": {
//         "ps": ["ps1:2222", "ps2:2222"],
//         "worker": ["worker1:2222", "worker2:2222", "worker3:2222"]
//     },
//     "task": {
//         "type": "ps",
//         "index": 1
//         },
//     }
// }
func genXDLConfigJSON(xdlJob *v1alpha1.XDLJob, rtype, index string) (string, error) {
	idx, err := strconv.ParseInt(index, 0, 32)
	if err != nil {
		return "", err
	}
	cluster, err := genClusterSpec(xdlJob)
	if err != nil {
		return "", err
	}
	xdlConfig := XDLConfig{
		Cluster: cluster,
		Task: TaskSpec{
			Type:  rtype,
			Index: int(idx),
		},
	}
	cfgJSON, err := json.Marshal(xdlConfig)
	if err != nil {
		return "", err
	}
	return string(cfgJSON), nil
}

// genClusterSpec will generate ClusterSpec.
func genClusterSpec(xdlJob *v1alpha1.XDLJob) (ClusterSpec, error) {
	clusterSpec := make(ClusterSpec)
	for rtype, spec := range xdlJob.Spec.XDLReplicaSpecs {
		rt := strings.ToLower(string(rtype))
		replicaNames := make([]string, 0, *spec.Replicas)

		port, err := job_controller.GetPortFromJob(xdlJob.Spec.XDLReplicaSpecs, rtype, v1alpha1.DefaultContainerName, v1alpha1.DefaultContainerPortName)
		if err != nil {
			return nil, err
		}
		for i := int32(0); i < *spec.Replicas; i++ {
			host := fmt.Sprintf("%s:%d", job_controller.GenGeneralName(xdlJob.Name, rt, fmt.Sprintf("%d", i)), port)
			replicaNames = append(replicaNames, host)
		}
		clusterSpec[rt] = replicaNames
	}
	return clusterSpec, nil
}
