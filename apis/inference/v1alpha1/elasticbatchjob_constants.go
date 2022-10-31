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
	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

const (
	ElasticBatchJobKind = "ElasticBatchJob"
	// ElasticBatchJobDefaultContainerName is the name of the ElasticBatchJob container.
	ElasticBatchJobDefaultContainerName = "elasticbatch"
	// ElasticBatchJobDefaultPortName is name of the port used to communicate between AIMaster and
	// workers.
	ElasticBatchJobDefaultPortName = "elstcbatch-port"
	// ElasticBatchJobDefaultPort is default value of the port.
	ElasticBatchJobDefaultPort = 23456
	// ElasticBatchJobDefaultMasterRestartPolicy is default RestartPolicy for Master ElasticBatchReplicaSpec.
	ElasticBatchJobDefaultAIMasterRestartPolicy = common.RestartPolicyNever
	// ElasticBatchJobDefaultWorkerRestartPolicy is default RestartPolicy for Worker ElasticBatchReplicaSpec,
	ElasticBatchJobDefaultWorkerRestartPolicy = common.RestartPolicyExitCode
)
