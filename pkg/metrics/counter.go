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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// JobCounter holds the kinds of metrics counter for some type of job workload.
type JobCounter struct {
	kind    string
	created *prometheus.CounterVec
	deleted *prometheus.CounterVec
	success *prometheus.CounterVec
	failure *prometheus.CounterVec
	restart *prometheus.CounterVec
}

func NewJobCounter(kind string) *JobCounter {
	counter := &JobCounter{
		kind: strings.ToLower(kind),
		created: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kubedl_jobs_created",
			Help: "Counts number of jobs created",
		}, []string{"kind"}),
		deleted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "controller_jobs_deleted",
			Help: "Counts number of jobs deleted",
		}, []string{"kind"}),
		success: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "controller_jobs_successful",
			Help: "Counts number of jobs successful",
		}, []string{"kind"}),
		failure: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "controller_jobs_failed",
			Help: "Counts number of jobs failed",
		}, []string{"kind"}),
		restart: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "controller_jobs_restarted",
			Help: "Counts number of jobs restarted",
		}, []string{"kind"}),
	}
	return counter
}

func (jc JobCounter) Created() prometheus.Counter {
	return jc.created.With(prometheus.Labels{"kind": jc.kind})
}

func (jc JobCounter) Deleted() prometheus.Counter {
	return jc.deleted.With(prometheus.Labels{"kind": jc.kind})
}

func (jc JobCounter) Success() prometheus.Counter {
	return jc.success.With(prometheus.Labels{"kind": jc.kind})
}

func (jc JobCounter) Failure() prometheus.Counter {
	return jc.failure.With(prometheus.Labels{"kind": jc.kind})
}

func (jc JobCounter) Restart() prometheus.Counter {
	return jc.restart.With(prometheus.Labels{"kind": jc.kind})
}
