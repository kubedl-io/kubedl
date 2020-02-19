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
	"fmt"

	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// GetJobFromInformerCache returns the Job from Informer Cache
func (r *TFJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &tfv1.TFJob{}
	// Default reader for TFJob is cache reader.
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "tf job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// GetJobFromAPIClient returns the Job from API server
func (r *TFJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &tfv1.TFJob{}
	// Forcibly use client reader.
	clientReader, err := util.GetClientReaderFromClient(r.Client)
	if err != nil {
		return nil, err
	}
	err = clientReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "tf job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// DeleteJob deletes the job
func (r *TFJobReconciler) DeleteJob(job interface{}) error {
	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of TFJob", tfJob)
	}
	if err := r.Delete(context.Background(), tfJob); err != nil {
		r.recorder.Eventf(tfJob, corev1.EventTypeWarning, job_controller.FailedDeleteJobReason, "Error deleting: %v", err)
		log.Error(err, "failed to delete job", "namespace", tfJob.Namespace, "name", tfJob.Name)
		return err
	}
	r.recorder.Eventf(tfJob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteJobReason, "Deleted job: %v", tfJob.Name)
	log.Info("job deleted", "namespace", tfJob.Namespace, "name", tfJob.Name)
	return nil
}

// UpdateJobStatus updates the job status and job conditions
func (r *TFJobReconciler) UpdateJobStatus(job interface{}, replicas map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of TFJob", tfJob)
	}
	return r.updateGeneralJobStatus(tfJob, replicas, jobStatus, restart)
}

// UpdateJobStatusInApiServer updates the job status in API server
func (r *TFJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *v1.JobStatus) error {
	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of TFJob", tfJob)
	}
	var jobCpy *tfv1.TFJob
	// Job status passed in differs with status in job, update in basis of the passed in one.
	jobCpy = tfJob.DeepCopy()
	jobCpy.Status = *jobStatus.DeepCopy()
	return r.Status().Update(context.Background(), jobCpy)
}
