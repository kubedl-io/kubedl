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

package metrics

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type JobGauge struct {
	running    *JobRunningGauge
	launchTime *LaunchTimeGauge
}

func NewJobGauge(kind string, reader client.Reader, interval time.Duration, runningCounter RunningCounterFunc) *JobGauge {
	return &JobGauge{
		running: NewJobRunningGauge(strings.ToLower(kind), prometheus.GaugeOpts{
			Name: "controller_running_jobs",
			Help: "Counts number of jobs running",
		}, reader, interval, runningCounter),
		launchTime: NewLaunchTimeGauge(kind, prometheus.GaugeOpts{
			Name: "controller_launch_time",
			Help: "Gauge for launch time of each job",
		}),
	}
}

func (jp JobGauge) Running() *JobRunningGauge {
	return jp.running
}

func (jp JobGauge) LaunchTime() *LaunchTimeGauge {
	return jp.launchTime
}
