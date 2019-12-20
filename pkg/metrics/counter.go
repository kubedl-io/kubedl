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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	created = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kubedl_jobs_created",
		Help: "Counts number of jobs created",
	}, []string{"kind"})
	deleted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kubedl_jobs_deleted",
		Help: "Counts number of jobs deleted",
	}, []string{"kind"})
	success = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kubedl_jobs_successful",
		Help: "Counts number of jobs successfully finished",
	}, []string{"kind"})
	failure = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kubedl_jobs_failed",
		Help: "Counts number of jobs failed",
	}, []string{"kind"})
	restart = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kubedl_jobs_restarted",
		Help: "Counts number of jobs restarted",
	}, []string{"kind"})
	running = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubedl_jobs_running",
		Help: "Counts number of jobs running currently",
	}, []string{"kind"})
	pending = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubedl_jobs_pending",
		Help: "Counts number of jobs pending currently",
	}, []string{"kind"})
)

// JobCounter holds the kinds of metrics counter for some type of job workload.
type JobCounter struct {
	statusCounter JobStatusCounterFunc
	created       prometheus.Counter
	deleted       prometheus.Counter
	success       prometheus.Counter
	failure       prometheus.Counter
	restart       prometheus.Counter
	running       prometheus.Gauge
	pending       prometheus.Gauge
}

func NewJobCounter(kind string, client client.Client) *JobCounter {
	kind = strings.ToLower(kind)
	label := prometheus.Labels{"kind": kind}
	counter := &JobCounter{
		statusCounter: JobStatusCounter(kind, client),
		created:       created.With(label),
		deleted:       deleted.With(label),
		success:       success.With(label),
		failure:       failure.With(label),
		restart:       restart.With(label),
		running:       running.With(label),
		pending:       pending.With(label),
	}
	return counter
}

func (jc *JobCounter) CreatedInc() {
	jc.created.Inc()
}

func (jc *JobCounter) DeletedInc() {
	jc.deleted.Inc()
}

func (jc *JobCounter) SuccessInc() {
	jc.success.Inc()
}

func (jc *JobCounter) FailureInc() {
	jc.failure.Inc()
}

func (jc *JobCounter) RestartInc() {
	jc.restart.Inc()
}

func (jc *JobCounter) PendingInc() {
	jc.pending.Inc()
}

func (jc *JobCounter) PendingDec() {
	jc.pending.Dec()
}

func (jc *JobCounter) RunningDec() {
	jc.running.Dec()
}

func (jc *JobCounter) RunningInc() {
	// Init number of currently running jobs in cluster, and this counter func
	// will only be invoked one time, then it will be set as nil.
	if jc.statusCounter != nil {
		running, pending, err := jc.statusCounter()
		if err == nil {
			jc.running.Set(float64(running))
			jc.pending.Set(float64(pending))
			jc.statusCounter = nil
			return
		}
	}
	jc.running.Inc()
}
