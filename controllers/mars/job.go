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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func (r *MarsJobReconciler) GetJobFromInformerCache(namespace, name string) (client.Object, error) {
	job := &v1alpha1.MarsJob{}
	// Default reader for PytorchJob is cache reader.
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("mars job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *MarsJobReconciler) GetJobFromAPIClient(namespace, name string) (client.Object, error) {
	job := &v1alpha1.MarsJob{}
	err := r.ctrl.APIReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("mars job not found", "namespace", namespace, "name", name)
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

func (r *MarsJobReconciler) UpdateJobStatus(job client.Object, replicas map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	marsJob, ok := job.(*v1alpha1.MarsJob)
	if !ok {
		return fmt.Errorf("%+v is not type of MarsJob", job)
	}
	return r.updateGeneralJobStatus(marsJob, replicas, jobStatus, restart)
}

func (r *MarsJobReconciler) UpdateJobStatusInApiServer(job client.Object, jobStatus *v1.JobStatus) error {
	marsJob, ok := job.(*v1alpha1.MarsJob)
	if !ok {
		return fmt.Errorf("%+v is not type of MarsJob", marsJob)
	}

	jobCpy := marsJob.DeepCopy()
	jobCpy.Status.JobStatus = *jobStatus.DeepCopy()
	return r.Status().Update(context.Background(), jobCpy)
}
