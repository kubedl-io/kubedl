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

package options

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/net"
)

var (
	CtrlConfig JobControllerConfiguration
)

// JobControllerConfiguration contains configuration of operator.
type JobControllerConfiguration struct {
	// Enable gang scheduling by abstract GangScheduler.
	EnableGangScheduling bool

	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run.
	// Defaults to 1.
	MaxConcurrentReconciles int

	// ReconcilerSyncLoopPeriod is the amount of time the reconciler sync states loop
	// wait between two reconciler sync.
	// It is set to 15 sec by default.
	// TODO(cph): maybe we can let it grows by multiple in the future
	// and up to 5 minutes to reduce idle loop.
	// e.g. 15s, 30s, 60s, 120s...
	ReconcilerSyncLoopPeriod v1.Duration

	// Name of global default gang scheduler.
	GangSchedulerName string

	// HostNetworkPortRange specifies host ports range to randomize for hostnetwork-enabled jobs.
	HostNetworkPortRange net.PortRange

	// The container builder image name, Kaniko image
	ModelImageBuilder string
}
