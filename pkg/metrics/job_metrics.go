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

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	firstPodLaunchDelayHist = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "kubedl_jobs_first_pod_launch_delay_seconds",
		Help: "Histogram for recording launch delay duration(from job created to first pod running).",
	}, []string{"kind", "name", "namespace", "uid"})
	allPodsLaunchDelayHist = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "kubedl_jobs_all_pods_launch_delay_seconds",
		Help: "Histogram for recording sync launch delay duration(from job created to all pods running).",
	}, []string{"kind", "name", "namespace", "uid"})
)

// JobMetrics holds the kinds of metrics counter for some type of job workload.
type JobMetrics struct {
	kind                string
	created             prometheus.Counter
	deleted             prometheus.Counter
	success             prometheus.Counter
	failure             prometheus.Counter
	restart             prometheus.Counter
	firstPodLaunchDelay *prometheus.HistogramVec
	allPodsLaunchDelay  *prometheus.HistogramVec
}

func NewJobMetrics(kind string, client client.Client) *JobMetrics {
	lowerKind := strings.ToLower(kind)
	label := prometheus.Labels{"kind": lowerKind}
	metrics := &JobMetrics{
		kind:                kind,
		created:             created.With(label),
		deleted:             deleted.With(label),
		success:             success.With(label),
		failure:             failure.With(label),
		restart:             restart.With(label),
		firstPodLaunchDelay: firstPodLaunchDelayHist,
		allPodsLaunchDelay:  allPodsLaunchDelayHist,
	}
	// Register running gauge func on center prometheus demand pull.
	// Different kinds of workload metrics share the same metric name and help info,
	// but const labels varies.
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "kubedl_jobs_running",
		Help:        "Counts number of jobs running currently",
		ConstLabels: label,
	}, func() float64 {
		running, err := JobStatusCounter(kind, client, util.IsRunning)
		if err != nil {
			return 0
		}
		return float64(running)
	})
	// Register pending gauge func on center prometheus demand pull.
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "kubedl_jobs_pending",
		Help:        "Counts number of jobs pending currently",
		ConstLabels: label,
	}, func() float64 {
		pending, err := JobStatusCounter(kind, client, func(status v1.JobStatus) bool {
			return util.IsCreated(status) && len(status.Conditions) == 1
		})
		if err != nil {
			return 0
		}
		return float64(pending)
	})
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

func (m *JobMetrics) FirstPodLaunchDelaySeconds(activePods []*corev1.Pod, job metav1.Object, status v1.JobStatus) {
	if !util.IsRunning(status) {
		return
	}

	var earliestTime *metav1.Time
	for _, pod := range activePods {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		readyCond := getPodCondition(&pod.Status, corev1.PodReady)
		if readyCond == nil {
			continue
		}
		if earliestTime == nil || readyCond.LastTransitionTime.Before(earliestTime) {
			earliestTime = &readyCond.LastTransitionTime
		}
	}
	if earliestTime == nil {
		return
	}
	delay := earliestTime.Time.Sub(job.GetCreationTimestamp().Time).Seconds()
	m.firstPodLaunchDelay.With(prometheus.Labels{
		"kind":      m.kind,
		"name":      job.GetName(),
		"namespace": job.GetNamespace(),
		"uid":       string(job.GetUID()),
	}).Observe(delay)
}

func (m *JobMetrics) AllPodsLaunchDelaySeconds(pods []*corev1.Pod, job metav1.Object, status v1.JobStatus) {
	if !util.IsRunning(status) || status.StartTime == nil {
		return
	}

	finalTime := job.GetCreationTimestamp().Time
	for _, pod := range pods {
		if pod.Status.Phase != corev1.PodRunning {
			return
		}
		readyCond := getPodCondition(&pod.Status, corev1.PodReady)
		if readyCond == nil {
			continue
		}
		if readyCond.LastTransitionTime.After(finalTime) {
			finalTime = readyCond.LastTransitionTime.Time
		}
	}
	syncDelay := finalTime.Sub(job.GetCreationTimestamp().Time).Seconds()
	m.allPodsLaunchDelay.With(prometheus.Labels{
		"kind":      m.kind,
		"name":      job.GetName(),
		"namespace": job.GetNamespace(),
		"uid":       string(job.GetUID()),
	}).Observe(syncDelay)
}

func getPodCondition(podStatus *corev1.PodStatus, condType corev1.PodConditionType) *corev1.PodCondition {
	for idx := range podStatus.Conditions {
		if podStatus.Conditions[idx].Type == condType {
			return &podStatus.Conditions[idx]
		}
	}
	return nil
}
