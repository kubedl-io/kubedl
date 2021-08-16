/*
Copyright 2021 The Alibaba Authors.

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

package mpi

import (
	"context"
	"fmt"

	mpiv1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	"github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *MPIJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
	mpiJob, ok := obj.(*mpiv1.MPIJob)
	if !ok {
		return nil, fmt.Errorf("%+v is not type of MPIJob", obj)
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(mpiJob.GetName()),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist, client.InNamespace(mpiJob.GetNamespace()), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	pods := util.ToPodPointerList(podlist.Items)
	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := job_controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := r.GetJobFromAPIClient(mpiJob.GetNamespace(), mpiJob.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != mpiJob.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", mpiJob.GetNamespace(), mpiJob.GetName(), fresh.GetUID(), mpiJob.GetUID())
		}
		return fresh, nil
	})
	cm := controller.NewPodControllerRefManager(job_controller.NewPodControl(r.Client, r.recorder), mpiJob, selector, r.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}
