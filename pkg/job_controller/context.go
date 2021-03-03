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

import v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

// context* defines a group of context keys for passing request-scoped data
// that transits routines.
const (
	// contextHostNetworkPorts is the key for passing selected host-ports, value is
	// a map object [replica-index: port].
	contextHostNetworkPorts = v1.KubeDLPrefix + "/hostnetwork-ports"
)
