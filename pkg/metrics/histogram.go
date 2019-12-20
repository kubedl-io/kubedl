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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	launchDelayHist = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "kubedl_job_launch_delay",
		Help: "Histogram for recording launch delay duration(from job created to job running) of each job.",
	}, []string{"kind", "name", "namespace", "uid"})
)

type JobHistogram struct {
	kind string
	hist *prometheus.HistogramVec
}

func NewJobHistogram(kind string) *JobHistogram {
	return &JobHistogram{
		kind: strings.ToLower(kind),
		hist: launchDelayHist,
	}
}

func (jh *JobHistogram) LaunchDelay(job metav1.Object, status v1.JobStatus) {
	cond := util.GetCondition(status, v1.JobRunning)
	if cond == nil {
		return
	}
	delay := cond.LastTransitionTime.Time.Sub(status.StartTime.Time).Seconds()
	jh.hist.With(prometheus.Labels{
		"kind":      jh.kind,
		"name":      job.GetName(),
		"namespace": job.GetNamespace(),
		"uid":       string(job.GetUID()),
	}).Observe(delay)
}
