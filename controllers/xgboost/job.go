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

	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// DeleteJob deletes the job
func (r *XgboostJobReconciler) DeleteJob(job interface{}) error {
	xgboostjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}
	if err := r.Delete(context.Background(), xgboostjob); err != nil {
		r.recorder.Eventf(xgboostjob, corev1.EventTypeWarning, job_controller.FailedDeleteJobReason, "Error deleting: %v", err)
		log.Error(err, "failed to delete job", "namespace", xgboostjob.Namespace, "name", xgboostjob.Name)
		return err
	}
	r.recorder.Eventf(xgboostjob, corev1.EventTypeNormal, job_controller.SuccessfulDeleteJobReason, "Deleted job: %v", xgboostjob.Name)
	log.Info("job deleted", "namespace", xgboostjob.Namespace, "name", xgboostjob.Name)
	return nil
}

// GetJobFromInformerCache returns the Job from Informer Cache
func (r *XgboostJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &v1alpha1.XGBoostJob{}
	// Default reader for XGBoostJob is cache reader.
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "xgboost job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// GetJobFromAPIClient returns the Job from API server
func (r *XgboostJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &v1alpha1.XGBoostJob{}

	clientReader, err := commonutil.GetClientReaderFromClient(r.Client)
	if err != nil {
		return nil, err
	}
	err = clientReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "xgboost job not found", "namespace", namespace, "name", name)
		} else {
			log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// UpdateJobStatus updates the job status and job conditions
func (r *XgboostJobReconciler) UpdateJobStatus(job interface{}, replicas map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	xgboostJob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of xgboostJob", xgboostJob)
	}

	previousRestarting := commonutil.IsRestarting(*jobStatus)
	previousFailed := commonutil.IsFailed(*jobStatus)

	for rtype, spec := range replicas {
		// If rtype in replica status not found, there must be a mistyped/invalid rtype in job spec,
		// and it has not been reconciled in previous processes, discard it.
		status, ok := jobStatus.ReplicaStatuses[rtype]
		if !ok {
			log.Info("skipping invalid replica type", "rtype", rtype)
			continue
		}

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logrus.Infof("XGBoostJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			xgboostJob.Name, rtype, expected, running, succeeded, failed)

		// All workers are running, update job start time.
		if status.Active == *spec.Replicas && jobStatus.StartTime == nil {
			now := metav1.Now()
			jobStatus.StartTime = &now
		}

		if rtype == v1alpha1.XGBoostReplicaTypeMaster {
			if running > 0 {
				msg := fmt.Sprintf("XGBoostJob %s is running.", xgboostJob.Name)
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
				if err != nil {
					log.Error(err, "Append job condition error")
					return err
				}
			}
			// when master is succeed, the job is finished.
			if expected == 0 {
				msg := fmt.Sprintf("XGBoostJob %s is successfully completed.", xgboostJob.Name)
				logrus.Info(msg)
				r.ctrl.Recorder.Event(xgboostJob, k8sv1.EventTypeNormal, commonutil.JobSucceededReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobSucceeded, commonutil.JobSucceededReason, msg)
				if err != nil {
					log.Error(err, "Append job condition error")
					return err
				}
				r.ctrl.Metrics.SuccessInc()
				return nil
			}
		}
		if failed > 0 {
			if restart {
				msg := fmt.Sprintf("XGBoostJob %s is restarting because %d %s replica(s) failed.", xgboostJob.Name, failed, rtype)
				r.ctrl.Recorder.Event(xgboostJob, k8sv1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					log.Error(err, "Append job condition error")
					return err
				}
				if !previousRestarting {
					r.ctrl.Metrics.FailureInc()
					r.ctrl.Metrics.RestartInc()
				}
			} else {
				msg := fmt.Sprintf("XGBoostJob %s is failed because %d %s replica(s) failed.", xgboostJob.Name, failed, rtype)
				r.ctrl.Recorder.Event(xgboostJob, k8sv1.EventTypeNormal, commonutil.JobFailedReason, msg)
				if xgboostJob.Status.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobFailed, commonutil.JobFailedReason, msg)
				if err != nil {
					log.Info("Append job condition", "error: ", err)
					return err
				}
				if !previousFailed {
					r.ctrl.Metrics.FailureInc()
				}
			}
		}
	}

	// Some workers are still running, leave a running condition.
	msg := fmt.Sprintf("XGBoostJob %s is running.", xgboostJob.Name)
	log.Info(msg)

	if err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg); err != nil {
		log.Error(err, "failed to update XGBoost Job conditions")
		return err
	}
	return nil
}

// UpdateJobStatusInApiServer updates the job status in to cluster.
func (r *XgboostJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *v1.JobStatus) error {
	xgboostjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}
	var jobCpy *v1alpha1.XGBoostJob
	// Job status passed in differs with status in job, update in basis of the passed in one.
	jobCpy = xgboostjob.DeepCopy()
	jobCpy.Status.JobStatus = *jobStatus.DeepCopy()
	return r.Status().Update(context.Background(), jobCpy)
}

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc(r reconcile.Reconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		xgboostJob, ok := e.Meta.(*v1alpha1.XGBoostJob)
		if !ok {
			return true
		}
		reconciler, ok := r.(*XgboostJobReconciler)
		if !ok {
			return true
		}
		reconciler.scheme.Default(xgboostJob)
		msg := fmt.Sprintf("xgboostJob %s is created.", e.Meta.GetName())
		if err := commonutil.UpdateJobConditions(&xgboostJob.Status.JobStatus, v1.JobCreated, commonutil.JobCreatedReason, msg); err != nil {
			log.Error(err, "append job condition error")
			return false
		}
		reconciler.ctrl.Metrics.CreatedInc()
		return true
	}
}
