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
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"
	"net/http"
	"strconv"
)

func StartMonitoringForDefaultRegistry(metricsPort int) {
	go func() {
		klog.Infof("setting up client for monitoring default registry on port: %s", strconv.Itoa(metricsPort))
		// Handler for default registry
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(metricsPort)), nil); err != nil {
			klog.Errorf("monitoring default registry failed, err:%v", err)
		}
	}()
}
