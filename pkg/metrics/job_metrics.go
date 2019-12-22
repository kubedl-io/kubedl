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
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	launchDelayHist = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "kubedl_jobs_launch_delay",
		Help: "Histogram for recording launch delay duration(from job created to job running) of each job.",
	}, []string{"kind", "name", "namespace", "uid"})
)

// JobMetrics holds the kinds of metrics counter for some type of job workload.
type JobMetrics struct {
	kind          string
	statusCounter JobStatusCounterFunc
	created       prometheus.Counter
	deleted       prometheus.Counter
	success       prometheus.Counter
	failure       prometheus.Counter
	restart       prometheus.Counter
	running       prometheus.Gauge
	pending       prometheus.Gauge
	launchDelay   *prometheus.HistogramVec
}

func NewJobMetrics(kind string, client client.Client) *JobMetrics {
	kind = strings.ToLower(kind)
	label := prometheus.Labels{"kind": kind}
	metrics := &JobMetrics{
		kind:          kind,
		statusCounter: JobStatusCounter(kind, client),
		created:       created.With(label),
		deleted:       deleted.With(label),
		success:       success.With(label),
		failure:       failure.With(label),
		restart:       restart.With(label),
		running:       running.With(label),
		pending:       pending.With(label),
		launchDelay:   launchDelayHist,
	}
	return metrics
}

func (m *JobMetrics) CreatedInc() {
	m.created.Inc()
}

func (m *JobMetrics) DeletedInc() {
	m.deleted.Inc()
}

func (m *JobMetrics) SuccessInc() {
	m.success.Inc()
}

func (m *JobMetrics) FailureInc() {
	m.failure.Inc()
}

func (m *JobMetrics) RestartInc() {
	m.restart.Inc()
}

func (m *JobMetrics) PendingInc() {
	m.pending.Inc()
}

func (m *JobMetrics) PendingDec() {
	m.pending.Dec()
}

func (m *JobMetrics) RunningDec() {
	m.running.Dec()
}

func (m *JobMetrics) RunningInc() {
	// Init number of currently running jobs in cluster, and this counter func
	// will only be invoked one time, then it will be set as nil.
	if m.statusCounter != nil {
		running, pending, err := m.statusCounter()
		if err == nil {
			m.running.Set(float64(running))
			m.pending.Set(float64(pending))
			m.statusCounter = nil
			return
		}
	}
	m.running.Inc()
}

func (m *JobMetrics) LaunchDelay(job metav1.Object, status v1.JobStatus) {
	cond := util.GetCondition(status, v1.JobRunning)
	if cond == nil {
		return
	}
	delay := metav1.Now().Time.Sub(status.StartTime.Time).Seconds()
	m.launchDelay.With(prometheus.Labels{
		"kind":      m.kind,
		"name":      job.GetName(),
		"namespace": job.GetNamespace(),
		"uid":       string(job.GetUID()),
	}).Observe(delay)
}
