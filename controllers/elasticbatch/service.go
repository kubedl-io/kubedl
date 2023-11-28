/*
Copyright 2012 The Alibaba Authors.

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

package elasticbatch

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
// i.e. all services created by the job will come with label "job-name" = <this_job_name>
func (r *ElasticBatchJobReconciler) GetServicesForJob(job client.Object) ([]*corev1.Service, error) {

	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(job.GetName()),
	})
	if err != nil {
		return nil, err
	}

	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	return r.ctrl.AdoptAndClaimServices(job, serviceList)
}
