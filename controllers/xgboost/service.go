/*

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

package xgboostjob

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

// CreateService creates the service
func (r *XgboostJobReconciler) CreateService(job interface{}, service *corev1.Service) error {
	xgboostjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	logrus.Info("Creating service ", " Controller name ", xgboostjob.GetName(), " Service name ", service.Namespace+"/"+service.Name)

	if err := r.Create(context.Background(), service); err != nil {
		logrus.Warnf("Create service error %s", xgboostjob.Name)
	}
	return nil
}

// DeleteService deletes the service
func (r *XgboostJobReconciler) DeleteService(job interface{}, name string, namespace string) error {
	xgboostjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}

	logrus.Info("Deleting service ", " Controller name ", xgboostjob.GetName(), " Service name ", service.Namespace+"/"+service.Name)

	if err := r.Delete(context.Background(), service); err != nil {
		if commonutil.IsSucceeded(xgboostjob.Status.JobStatus) {
			r.recorder.Eventf(xgboostjob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteServiceReason, "Deleted service: %v", name)
			return nil
		}

		r.recorder.Eventf(xgboostjob, corev1.EventTypeWarning, job_controller.FailedDeleteServiceReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete service: %v", err)
	}

	r.recorder.Eventf(xgboostjob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteServiceReason, "Deleted service: %v", name)

	return nil

}

// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
// i.e. all services created by the job will come with label "job-name" = <this_job_name>
func (r *XgboostJobReconciler) GetServicesForJob(obj interface{}) ([]*corev1.Service, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, fmt.Errorf("%+v is not a type of XGBoostJob", job)
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
	services := commonutil.ToServicePointerList(serviceList.Items)
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
