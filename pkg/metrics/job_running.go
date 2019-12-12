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
	"context"
	"sync"
	"time"

	pytorchv1 "github.com/alibaba/kubedl/api/pytorch/v1"
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// JobRunningGauge report number of running jobs currently in cluster. It will be triggered
// each interval duration.
type JobRunningGauge struct {
	kind           string
	gauge          *prometheus.GaugeVec
	reader         client.Reader
	interval       time.Duration
	runningCounter RunningCounterFunc

	lock       sync.RWMutex
	lastReport time.Time
}

// RunningCounterFunc list some kind of job in cluster, and counts the
// number of running jobs among listed one.
type RunningCounterFunc func(reader client.Reader) (int, error)

func NewTFJobRunningGauge(kind string, opts prometheus.GaugeOpts, reader client.Reader, interval time.Duration) *JobRunningGauge {
	return NewJobRunningGauge(kind, opts, reader, interval, TFJobRunningCounter)
}

func NewXDLJobRunningGauge(kind string, opts prometheus.GaugeOpts, reader client.Reader, interval time.Duration) *JobRunningGauge {
	return NewJobRunningGauge(kind, opts, reader, interval, XDLJobRunningCounter)
}

func NewPytorchJobRunningGauge(kind string, opts prometheus.GaugeOpts, reader client.Reader, interval time.Duration) *JobRunningGauge {
	return NewJobRunningGauge(kind, opts, reader, interval, PytorchJobRunningCounter)
}

func NewXGBoostJobRunningGauge(kind string, opts prometheus.GaugeOpts, reader client.Reader, interval time.Duration) *JobRunningGauge {
	return NewJobRunningGauge(kind, opts, reader, interval, XGBoostJobRunningCounter)
}

func NewJobRunningGauge(kind string, opts prometheus.GaugeOpts, reader client.Reader, interval time.Duration, runningCounter RunningCounterFunc) *JobRunningGauge {
	gauge := &JobRunningGauge{
		kind:           kind,
		gauge:          prometheus.NewGaugeVec(opts, []string{"kind"}),
		reader:         reader,
		interval:       interval,
		lastReport:     time.Now(),
		runningCounter: runningCounter,
	}
	prometheus.MustRegister(gauge.gauge)
	go gauge.run()
	return gauge
}

func (g *JobRunningGauge) run() {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()
	// Ticker will check gauge status each interval duration, if gauging is not recent
	// active, ticker will do gauging instantly.
	for {
		select {
		case <-ticker.C:
			if !g.recentActive() {
				g.Gauge()
			}
		}
	}
}

func (g *JobRunningGauge) updateReportTime() {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.lastReport = time.Now()
}

func (g *JobRunningGauge) lastReportTime() time.Time {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.lastReport
}

func (g *JobRunningGauge) recentActive() bool {
	return time.Now().Sub(g.lastReportTime()) < g.interval
}

func (g *JobRunningGauge) Gauge() {
	// If report has been triggered in last `interval` duration, then skip it.
	// Running job gauge should not call too frequently.
	if g.recentActive() {
		return
	}
	running, err := g.runningCounter(g.reader)
	if err != nil {
		logrus.Errorf("failed to count running jobs, err=%v", err)
		return
	}
	g.updateReportTime()
	g.gauge.With(prometheus.Labels{"kind": g.kind}).Set(float64(running))
}

func TFJobRunningCounter(reader client.Reader) (int, error) {
	tfJobList := &tfv1.TFJobList{}
	err := reader.List(context.Background(), tfJobList)
	if err != nil {
		return 0, err
	}
	running := 0
	for _, job := range tfJobList.Items {
		if util.IsRunning(job.Status) {
			running++
		}
	}
	return running, nil
}

func XDLJobRunningCounter(reader client.Reader) (int, error) {
	xdlJobList := &xdlv1alpha1.XDLJobList{}
	err := reader.List(context.Background(), xdlJobList)
	if err != nil {
		return 0, err
	}
	running := 0
	for _, job := range xdlJobList.Items {
		if util.IsRunning(job.Status) {
			running++
		}
	}
	return running, nil
}

func PytorchJobRunningCounter(reader client.Reader) (int, error) {
	pytorchJobList := &pytorchv1.PyTorchJobList{}
	err := reader.List(context.Background(), pytorchJobList)
	if err != nil {
		return 0, err
	}
	running := 0
	for _, job := range pytorchJobList.Items {
		if util.IsRunning(job.Status) {
			running++
		}
	}
	return running, nil
}

func XGBoostJobRunningCounter(reader client.Reader) (int, error) {
	xgboostJobList := &v1alpha1.XGBoostJobList{}
	err := reader.List(context.Background(), xgboostJobList)
	if err != nil {
		return 0, err
	}
	running := 0
	for _, job := range xgboostJobList.Items {
		if util.IsRunning(job.Status.JobStatus) {
			running++
		}
	}
	return running, nil
}
