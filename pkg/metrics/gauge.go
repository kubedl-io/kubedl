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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	launchTimeGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubedl_launch_time",
		Help: "Gauge for launch time of each job",
	}, []string{"kind", "name", "namespace", "uid"})
)

type JobGauge struct {
	launchTime *LaunchTimeGauge
}

func NewJobGauge(kind string) *JobGauge {
	return &JobGauge{
		launchTime: NewLaunchTimeGauge(kind, launchTimeGauge),
	}
}

func (jp JobGauge) LaunchTime() *LaunchTimeGauge {
	return jp.launchTime
}
