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
	common "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

const (
	PyTorchJobKind = "PyTorchJob"
	// PyTorchJobDefaultPortName is name of the port used to communicate between Master and
	// workers.
	PyTorchJobDefaultPortName = "pytorchjob-port"
	// PyTorchJobDefaultContainerName is the name of the PyTorchJob container.
	PyTorchJobDefaultContainerName = "pytorch"
	// PyTorchJobDefaultPort is default value of the port.
	PyTorchJobDefaultPort = 23456
	// PyTorchJobDefaultMasterRestartPolicy is default RestartPolicy for Master PyTorchReplicaSpec.
	PyTorchJobDefaultMasterRestartPolicy = common.RestartPolicyExitCode
	// PyTorchJobDefaultWorkerRestartPolicy is default RestartPolicy for Worker PyTorchReplicaSpec,
	PyTorchJobDefaultWorkerRestartPolicy = common.RestartPolicyOnFailure
)
