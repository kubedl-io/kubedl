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

package pytorch

import (
	"fmt"
	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func ContainMasterSpec(job *training.PyTorchJob) bool {
	_, ok := job.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeMaster]
	return ok
}

// computeDesiredReplicas retrieve user's replica setting in specs
func computeDesiredReplicas(elasticJob *training.PyTorchJob) (int32, error) {
	workerSpecs, exist := elasticJob.Spec.PyTorchReplicaSpecs[v1.ReplicaType(training.PyTorchReplicaTypeMaster)]
	if !exist {
		return 0, fmt.Errorf("elasticJob %v doesn't have %s", elasticJob, training.PyTorchReplicaTypeMaster)
	}

	return *workerSpecs.Replicas, nil
}
