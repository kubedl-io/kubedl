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

	pytorchv1 "github.com/alibaba/kubedl/api/pytorch/v1"
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	"github.com/alibaba/kubedl/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RunningCounterFunc list some kind of job in cluster, and counts the
// number of running jobs among listed one.
type RunningCounterFunc func() (int32, error)

func TFJobRunningCounter(reader client.Reader) RunningCounterFunc {
	return func() (i int32, err error) {
		tfJobList := &tfv1.TFJobList{}
		err = reader.List(context.Background(), tfJobList)
		if err != nil {
			return 0, err
		}
		running := int32(0)
		for _, job := range tfJobList.Items {
			if util.IsRunning(job.Status) {
				running++
			}
		}
		return running, nil
	}
}

func XDLJobRunningCounter(reader client.Reader) RunningCounterFunc {
	return func() (i int32, err error) {
		xdlJobList := &xdlv1alpha1.XDLJobList{}
		err = reader.List(context.Background(), xdlJobList)
		if err != nil {
			return 0, err
		}
		running := int32(0)
		for _, job := range xdlJobList.Items {
			if util.IsRunning(job.Status) {
				running++
			}
		}
		return running, nil
	}
}

func PytorchJobRunningCounter(reader client.Reader) RunningCounterFunc {
	return func() (i int32, err error) {
		pytorchJobList := &pytorchv1.PyTorchJobList{}
		err = reader.List(context.Background(), pytorchJobList)
		if err != nil {
			return 0, err
		}
		running := int32(0)
		for _, job := range pytorchJobList.Items {
			if util.IsRunning(job.Status) {
				running++
			}
		}
		return running, nil
	}
}

func XGBoostJobRunningCounter(reader client.Reader) RunningCounterFunc {
	return func() (i int32, err error) {
		xgboostJobList := &v1alpha1.XGBoostJobList{}
		err = reader.List(context.Background(), xgboostJobList)
		if err != nil {
			return 0, err
		}
		running := int32(0)
		for _, job := range xgboostJobList.Items {
			if util.IsRunning(job.Status.JobStatus) {
				running++
			}
		}
		return running, nil
	}
}
