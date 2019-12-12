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

package xdljob

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	"github.com/alibaba/kubedl/pkg/util"
)

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *XDLJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(job.GetName()),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	pods := util.ToPodPointerList(podlist.Items)
	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := job_controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := r.GetJobFromAPIClient(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := controller.NewPodControllerRefManager(job_controller.NewPodControl(r.Client, r.recorder), job, selector, r.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}

// CreatePod creates the pod
func (r *XDLJobReconciler) CreatePod(job interface{}, pod *corev1.Pod) error {
	return r.Create(context.Background(), pod)
}

// DeletePod deletes the pod
func (r *XDLJobReconciler) DeletePod(job interface{}, pod *corev1.Pod) error {
	xdlJob, ok := job.(*xdlv1alpha1.XDLJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XDLJob", job)
	}

	log.Info("Deleting pod", "controller name", xdlJob.GetName(), "pod name", pod.Namespace+"/"+pod.Name)
	if err := r.Delete(context.Background(), pod); err != nil && !errors.IsNotFound(err) {
		r.recorder.Eventf(xdlJob, corev1.EventTypeWarning, job_controller.FailedDeletePodReason, "Error deleting: %v", err)
		return err
	}
	r.recorder.Eventf(xdlJob, corev1.EventTypeNormal, job_controller.SuccessfulDeletePodReason, "Deleted pod: %v", pod.Name)
	return nil
}
