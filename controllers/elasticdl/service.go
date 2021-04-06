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

package elasticdl

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
)

// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
// i.e. all services created by the job will come with label "job-name" = <this_job_name>
func (r *ElasticDLJobReconciler) GetServicesForJob(obj interface{}) ([]*corev1.Service, error) {
	return []*corev1.Service{}, nil
}

// CreateService creates the service
func (r *ElasticDLJobReconciler) CreateService(job interface{}, service *corev1.Service) error {
	return r.Create(context.Background(), service)
}

// DeleteService deletes the service
func (r *ElasticDLJobReconciler) DeleteService(job interface{}, name string, namespace string) error {
	elasticdlJob, ok := job.(*training.ElasticDLJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of ElasticDLJob", job)
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
	log.Info("Deleting service", "controller name", r.ControllerName(), "service name", namespace+"/"+name)
	if err := r.Delete(context.Background(), service); err != nil && !errors.IsNotFound(err) {
		r.recorder.Eventf(elasticdlJob, corev1.EventTypeWarning, job_controller.FailedDeleteServiceReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete service: %v", err)
	}
	r.recorder.Eventf(elasticdlJob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteServiceReason, "Deleted service: %v", name)
	return nil
}
