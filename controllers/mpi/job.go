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

	"sigs.k8s.io/controller-runtime/pkg/client"

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *MPIJobReconciler) GetJobFromInformerCache(namespace, name string) (client.Object, error) {
	job := &training.MPIJob{}
	// Default reader for MPIJob is cache reader.
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("mpi job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *MPIJobReconciler) GetJobFromAPIClient(namespace, name string) (client.Object, error) {
	job := &training.MPIJob{}
	err := r.ctrl.APIReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("mpi job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *MPIJobReconciler) DeleteJob(job interface{}) error {
	mpiJob, ok := job.(*training.MPIJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of MPIJob", mpiJob)
	}
	if err := r.Delete(context.Background(), mpiJob); err != nil {
		r.recorder.Eventf(mpiJob, corev1.EventTypeWarning, job_controller.FailedDeleteJobReason, "Error deleting: %v", err)
		log.Error(err, "failed to delete job", "namespace", mpiJob.Namespace, "name", mpiJob.Name)
		return err
	}
	r.recorder.Eventf(mpiJob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteJobReason, "Deleted job: %v", mpiJob.Name)
	log.Info("job deleted", "namespace", mpiJob.Namespace, "name", mpiJob.Name)
	return nil
}

func (r *MPIJobReconciler) UpdateJobStatus(job client.Object, replicas map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	mpiJob, ok := job.(*training.MPIJob)
	if !ok {
		return fmt.Errorf("%+v is not type of MPIJob", job)
	}

	previousRestarting := util.IsRestarting(*jobStatus)
	previousFailed := util.IsFailed(*jobStatus)
	launcherStatus := jobStatus.ReplicaStatuses[training.MPIReplicaTypeLauncher]
	workerStatus := jobStatus.ReplicaStatuses[training.MPIReplicaTypeWorker]

	if launcherStatus != nil {
		if launcherStatus.Succeeded > 0 {
			msg := fmt.Sprintf("MPIJob %s has successfully completed.", mpiJob.Name)
			r.recorder.Eventf(mpiJob, corev1.EventTypeNormal, util.JobSucceededReason, msg)
			if jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}
			if err := util.UpdateJobConditions(jobStatus, v1.JobSucceeded, util.JobSucceededReason, msg); err != nil {
				log.Error(err, "failed to update mpi job conditions")
				return err
			}
			r.ctrl.Metrics.SuccessInc()
			return nil
		} else if launcherStatus.Failed > 0 {
			msg := fmt.Sprintf("MPIJob %s is failed because %d Launcher replica(s) failed", mpiJob.Name, launcherStatus.Failed)
			reason := util.JobFailedReason
			r.recorder.Eventf(mpiJob, corev1.EventTypeNormal, reason, msg)

			// Expose evicted reason in job conditions, and skip setting completion time
			// for a evicted job.
			if launcherStatus.Evicted > 0 {
				reason = util.JobEvictedReason
			} else if !util.IsEvicted(mpiJob.Status) && jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}

			if err := util.UpdateJobConditions(jobStatus, v1.JobFailed, reason, msg); err != nil {
				log.Error(err, "failed to update mpi job conditions")
				return err
			}
			if !previousFailed {
				r.ctrl.Metrics.FailureInc()
			}
		}
	}

	if workerStatus != nil {
		workerReplicas := *replicas[training.MPIReplicaTypeWorker].Replicas
		// Append and broadcast Evict event if evicted workers occurs.
		if workerStatus.Evicted > 0 {
			msg := fmt.Sprintf("%d/%d workers are evicted.", workerStatus.Evicted, workerReplicas)
			log.Info(msg, "MPIJob", mpiJob.Namespace+"/"+mpiJob.Name)
			if err := util.UpdateJobConditions(jobStatus, v1.JobFailed, util.JobEvictedReason, msg); err != nil {
				log.Error(err, "failed to update mpi job conditions")
				return err
			}
			r.recorder.Eventf(mpiJob, corev1.EventTypeNormal, util.JobRunningReason, "MPIJob %s/%s is running", mpiJob.Namespace, mpiJob.Name)
		}
		// Append and broadcast Restarting event when partial workers failed but retryable.
		if workerStatus.Failed > 0 && restart {
			msg := fmt.Sprintf("MPIJob %s is restarting because %d Worker replica(s) failed", mpiJob.Name, workerReplicas)
			r.recorder.Eventf(mpiJob, corev1.EventTypeWarning, util.JobRestartingReason, msg)
			if err := util.UpdateJobConditions(jobStatus, v1.JobRestarting, util.JobRestartingReason, msg); err != nil {
				log.Error(err, "failed to update mpi job conditions")
				return err
			}
			if !previousRestarting {
				r.ctrl.Metrics.FailureInc()
				r.ctrl.Metrics.RestartInc()
			}
		} else if launcherStatus != nil && launcherStatus.Active > 0 && workerStatus.Active == workerReplicas {
			// Job is marked as Running only when launcher was active and all workers has been in running state.
			msg := fmt.Sprintf("MPIJob %s is running.", mpiJob.Name)
			err := util.UpdateJobConditions(jobStatus, v1.JobRunning, util.JobRunningReason, msg)
			if err != nil {
				log.Info("Append job condition", " error:", err)
				return err
			}
		}
	}
	return nil
}

func (r *MPIJobReconciler) UpdateJobStatusInApiServer(job client.Object, jobStatus *v1.JobStatus) error {
	mpiJob, ok := job.(*training.MPIJob)
	if !ok {
		return fmt.Errorf("%+v is not type of MarsJob", mpiJob)
	}

	jobCpy := mpiJob.DeepCopy()
	jobCpy.Status = *jobStatus.DeepCopy()
	return r.Status().Update(context.Background(), jobCpy)
}
