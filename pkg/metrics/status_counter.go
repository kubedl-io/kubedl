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

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	inference1alpha1 "github.com/alibaba/kubedl/apis/inference/v1alpha1"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func JobStatusCounter(kind string, reader client.Reader, filter func(status v1.JobStatus) bool) (result int32, err error) {
	var list client.ObjectList
	if obj, ok := listObjectMap[kind]; ok {
		list = obj.DeepCopyObject().(client.ObjectList)
	}
	err = reader.List(context.Background(), list)
	if err != nil {
		return 0, err
	}
	statuses := getJobStatusList(list, kind)
	result = int32(0)
	for _, status := range statuses {
		if filter(*status) {
			result++
		}
	}
	return result, nil
}

var (
	listObjectMap = map[string]client.ObjectList{
		trainingv1alpha1.TFJobKind:           &trainingv1alpha1.TFJobList{},
		trainingv1alpha1.PyTorchJobKind:      &trainingv1alpha1.PyTorchJobList{},
		trainingv1alpha1.XDLJobKind:          &trainingv1alpha1.XDLJobList{},
		trainingv1alpha1.XGBoostJobKind:      &trainingv1alpha1.XGBoostJobList{},
		trainingv1alpha1.MarsJobKind:         &trainingv1alpha1.MarsJobList{},
		inference1alpha1.ElasticBatchJobKind: &inference1alpha1.ElasticBatchJobList{},
	}
)

func getJobStatusList(obj runtime.Object, kind string) []*v1.JobStatus {
	statuses := make([]*v1.JobStatus, 0)
	switch kind {
	case trainingv1alpha1.TFJobKind:
		tfList := obj.(*trainingv1alpha1.TFJobList)
		for idx := range tfList.Items {
			statuses = append(statuses, &tfList.Items[idx].Status)
		}
	case trainingv1alpha1.PyTorchJobKind:
		pytorchList := obj.(*trainingv1alpha1.PyTorchJobList)
		for idx := range pytorchList.Items {
			statuses = append(statuses, &pytorchList.Items[idx].Status)
		}
	case trainingv1alpha1.XDLJobKind:
		xdlList := obj.(*trainingv1alpha1.XDLJobList)
		for idx := range xdlList.Items {
			statuses = append(statuses, &xdlList.Items[idx].Status)
		}
	case trainingv1alpha1.XGBoostJobKind:
		xgbList := obj.(*trainingv1alpha1.XGBoostJobList)
		for idx := range xgbList.Items {
			statuses = append(statuses, &xgbList.Items[idx].Status.JobStatus)
		}
	}
	return statuses
}
