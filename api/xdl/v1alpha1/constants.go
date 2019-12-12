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
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

const (
	// Field of XDLJobSpec, 0 indicate that job finish util all workers done.
	DefaultMinFinishWorkNum int32 = 0
	// Field of XDLJobSpec, 90 indicate that job finish util 90% workers done.
	DefaultMinFinishWorkRate int32 = 90
	// Field of XDLJobSpec, default total failover times of job is 20.
	DefaultBackoffLimit int32 = 20
	// TODO(qiukai.cqk): ensure default names
	DefaultContainerName     = "xdl"
	DefaultContainerPortName = "xdljob-port"
	DefaultPort              = 2222
	DefaultRestartPolicy     = v1.RestartPolicyNever
)
