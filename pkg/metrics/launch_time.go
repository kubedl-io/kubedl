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
	commonv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// LaunchTimeGauge reports launch time(from acknowledged by controller to first worker running)
// of each tf job.
type LaunchTimeGauge struct {
	kind  string
	gauge *prometheus.GaugeVec
	// runningSet to cache uid of jobs who has reported when come into running state.
	runningSet sets.String
}

func NewLaunchTimeGauge(kind string, opts prometheus.GaugeOpts) *LaunchTimeGauge {
	gauge := &LaunchTimeGauge{
		kind:       kind,
		gauge:      prometheus.NewGaugeVec(opts, []string{"kind", "name", "namespace", "uid"}),
		runningSet: sets.NewString(),
	}
	prometheus.MustRegister(gauge.gauge)
	return gauge
}

func (g *LaunchTimeGauge) jobGauged(obj v1.Object, status commonv1.JobStatus) bool {
	var gauged bool
	uid := string(obj.GetUID())
	gauged = g.runningSet.Has(uid)
	if util.IsSucceeded(status) || util.IsFailed(status) {
		// Job has finished, remove it from cache.
		g.runningSet.Delete(uid)
	}
	return gauged
}

func (g *LaunchTimeGauge) Gauge(obj v1.Object, status commonv1.JobStatus) {
	// Launch time should gauge once in the whole lifecycle of job.
	if g.jobGauged(obj, status) {
		return
	}
	found := false
	var runningCond commonv1.JobCondition
	// Get job running condition information if it has successfully migrated to running state.
	for _, cond := range status.Conditions {
		if cond.Type == commonv1.JobRunning && cond.Status == corev1.ConditionTrue {
			found = true
			runningCond = cond
			break
		}
	}
	if !found {
		return
	}
	launchSeconds := runningCond.LastTransitionTime.Time.
		Sub(status.StartTime.Time).
		Seconds()
	g.gauge.With(prometheus.Labels{
		"kind":      g.kind,
		"name":      obj.GetName(),
		"namespace": obj.GetNamespace(),
		"uid":       string(obj.GetUID()),
	}).Set(launchSeconds)
	// Add job into cache with uid.
	g.runningSet.Insert(string(obj.GetUID()))
}
