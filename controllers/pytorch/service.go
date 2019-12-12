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

package pytorch

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pytorchv1 "github.com/alibaba/kubedl/api/pytorch/v1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	"github.com/alibaba/kubedl/pkg/util"
)

// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
// i.e. all services created by the job will come with label "job-name" = <this_job_name>
func (r *PytorchJobReconciler) GetServicesForJob(obj interface{}) ([]*corev1.Service, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(job.GetName()),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	services := util.ToServicePointerList(serviceList.Items)
	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing services (see #42639).
	canAdoptFunc := job_controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := r.GetJobFromInformerCache(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := job_controller.NewServiceControllerRefManager(job_controller.NewServiceControl(r.Client, r.recorder), job, selector, r.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimServices(services)
}

// CreateService creates the service
func (r *PytorchJobReconciler) CreateService(job interface{}, service *corev1.Service) error {
	return r.Create(context.Background(), service)
}

// DeleteService deletes the service
func (r *PytorchJobReconciler) DeleteService(job interface{}, name string, namespace string) error {
	pytorchJob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PytorchJob", job)
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
	log.Info("Deleting service", "controller name", r.ControllerName(), "service name", namespace+"/"+name)
	if err := r.Delete(context.Background(), service); err != nil && !errors.IsNotFound(err) {
		r.recorder.Eventf(pytorchJob, corev1.EventTypeWarning, job_controller.FailedDeleteServiceReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete service: %v", err)
	}
	r.recorder.Eventf(pytorchJob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteServiceReason, "Deleted service: %v", name)
	return nil
}
