/*
Copyright 2020 The Alibaba Authors.

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

package util

import (
	inferencev1alpha1 "github.com/alibaba/kubedl/apis/inference/v1alpha1"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
)

func IsKubeDLManagedJobKind(kind string) bool {
	return kind == trainingv1alpha1.TFJobKind || kind == trainingv1alpha1.PyTorchJobKind ||
		kind == trainingv1alpha1.XDLJobKind || kind == trainingv1alpha1.XGBoostJobKind || kind == inferencev1alpha1.ElasticBatchJobKind
}

func IsKubeDLManagedPod(pod *corev1.Pod) bool {
	controller := util.GetControllerOwnerReference(pod.OwnerReferences)
	if !IsKubeDLManagedJobKind(controller.Kind) {
		return false
	}
	_, ok := pod.Labels[apiv1.GroupNameLabel]
	return ok
}
