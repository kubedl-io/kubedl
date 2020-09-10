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

package controllers

import (
	"context"
	"fmt"

	"github.com/alibaba/kubedl/api/marsjob/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *MarsJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &v1alpha1.MarsJob{}
	// Default reader for PytorchJob is cache reader.
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "mars job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *MarsJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &v1alpha1.MarsJob{}
	// Forcibly use client reader.
	clientReader, err := util.GetClientReaderFromClient(r.Client)
	if err != nil {
		return nil, err
	}
	err = clientReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "mars job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *MarsJobReconciler) DeleteJob(job interface{}) error {
	marsJob, ok := job.(*v1alpha1.MarsJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of MarsJob", marsJob)
	}
	if err := r.Delete(context.Background(), marsJob); err != nil {
		r.recorder.Eventf(marsJob, corev1.EventTypeWarning, job_controller.FailedDeleteJobReason, "Error deleting: %v", err)
		log.Error(err, "failed to delete job", "namespace", marsJob.Namespace, "name", marsJob.Name)
		return err
	}
	r.recorder.Eventf(marsJob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteJobReason, "Deleted job: %v", marsJob.Name)
	log.Info("job deleted", "namespace", marsJob.Namespace, "name", marsJob.Name)
	return nil
}

func (r *MarsJobReconciler) UpdateJobStatus(job interface{}, replicas map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	marsJob, ok := job.(*v1alpha1.MarsJob)
	if !ok {
		return fmt.Errorf("%+v is not type of MarsJob", job)
	}
	return r.updateGeneralJobStatus(marsJob, replicas, jobStatus, restart)
}

func (r *MarsJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *v1.JobStatus) error {
	marsJob, ok := job.(*v1alpha1.MarsJob)
	if !ok {
		return fmt.Errorf("%+v is not type of MarsJob", marsJob)
	}

	var jobCpy *v1alpha1.MarsJob
	jobCpy = marsJob.DeepCopy()
	jobCpy.Status = *jobStatus.DeepCopy()
	return r.Status().Update(context.Background(), jobCpy)
}
