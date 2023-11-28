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

package tensorflow

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodsForJob returns the set of pods that this job should manage.
func (r *TFJobReconciler) GetPodsForJob(job client.Object) ([]*corev1.Pod, error) {
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(job.GetName()),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podList := &corev1.PodList{}
	err := r.List(context.Background(), podList, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	return r.ctrl.AdoptAndClaimPods(job, podList)
}
