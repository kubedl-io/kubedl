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
	"fmt"
	"math"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

const (
	xdlJobFailedReason     = "XdlJobFailed"
	xdlJobRestartingReason = "XdlJobRestarting"
)

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc(r reconcile.Reconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		xdlJob, ok := e.Meta.(*xdlv1alpha1.XDLJob)
		if !ok {
			return true
		}
		reconciler, ok := r.(*XDLJobReconciler)
		if !ok {
			return true
		}
		reconciler.scheme.Default(xdlJob)
		msg := fmt.Sprintf("XdlJob %s is created.", e.Meta.GetName())
		if err := commonutil.UpdateJobConditions(&xdlJob.Status, v1.JobCreated, commonutil.JobCreatedReason, msg); err != nil {
			log.Error(err, "append job condition error")
			return false
		}
		reconciler.ctrl.Metrics.CreatedInc()
		return true
	}
}

// updateGeneralJobStatus updates the status of job with given replica specs and job status.
func (r *XDLJobReconciler) updateGeneralJobStatus(xdlJob *xdlv1alpha1.XDLJob, replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec,
	jobStatus *v1.JobStatus, restart bool) error {
	log.Info("Updating status", "XDLJob name", xdlJob.Name)

	previousRestarting := commonutil.IsRestarting(*jobStatus)
	previousFailed := commonutil.IsFailed(*jobStatus)

	workerNum, workerSucceeded := int32(0), int32(0)
	// Iterate all replicas for counting and try to update start time.
	for rtype, spec := range replicaSpecs {
		replicas := *spec.Replicas
		// If rtype in replica status not found, there must be a mistyped/invalid rtype in job spec,
		// and it has not been reconciled in previous processes, discard it.
		status, ok := jobStatus.ReplicaStatuses[rtype]
		if !ok {
			log.Info("skipping invalid replica type", "rtype", rtype)
			continue
		}
		failed := status.Failed
		if rtype == xdlv1alpha1.XDLReplicaTypeWorker || rtype == xdlv1alpha1.XDLReplicaTypeExtendRole {
			workerNum += replicas
			workerSucceeded += status.Succeeded
		}
		// All workers are running, update job start time.
		if status.Active == replicas && jobStatus.StartTime == nil {
			now := metav1.Now()
			jobStatus.StartTime = &now
		}

		if failed > 0 {
			if restart {
				msg := fmt.Sprintf("XDLJob %s is restarting because %d %s replica(s) failed.", xdlJob.Name, failed, rtype)
				r.recorder.Event(xdlJob, corev1.EventTypeWarning, xdlJobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					log.Error(err, "failed to update xdl job conditions")
					return err
				}
				if !previousRestarting {
					r.ctrl.Metrics.FailureInc()
					r.ctrl.Metrics.RestartInc()
				}
			} else {
				msg := fmt.Sprintf("XDLJob %s is failed because %d %s replica(s) failed.", xdlJob.Name, failed, rtype)
				r.recorder.Event(xdlJob, corev1.EventTypeNormal, xdlJobFailedReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobFailed, commonutil.JobFailedReason, msg)
				if err != nil {
					log.Error(err, "failed to update xdl job conditions")
					return err
				}
				if !previousFailed {
					r.ctrl.Metrics.FailureInc()
				}
			}
			return nil
		}
	}

	workerMinFinish := calculateMinFinish(xdlJob, workerNum)

	// Number of succeeded workers has reached the threshold, job successfully completed.
	if workerSucceeded >= workerMinFinish {
		msg := fmt.Sprintf("XDLJob %s is successfully completed.", xdlJob.Name)
		if jobStatus.CompletionTime == nil {
			now := metav1.Now()
			jobStatus.CompletionTime = &now
		}
		if err := commonutil.UpdateJobConditions(jobStatus, v1.JobSucceeded, commonutil.JobSucceededReason, msg); err != nil {
			log.Error(err, "failed to update xdl job conditions")
			return err
		}
		r.ctrl.Metrics.SuccessInc()
		return nil
	}

	// Some workers are still running, leave a running condition.
	msg := fmt.Sprintf("XDLJob %s is running.", xdlJob.Name)
	if err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg); err != nil {
		log.Error(err, "failed to update xdl job conditions")
		return err
	}
	return nil
}

// calculateMinFinish calculate for minimal workers finished workers such that the job
// is treated as successful.
func calculateMinFinish(xdlJob *xdlv1alpha1.XDLJob, workersNum int32) int32 {
	if xdlJob.Spec.MinFinishWorkerPercentage != nil {
		return int32(math.Ceil(float64(workersNum*(*xdlJob.Spec.MinFinishWorkerPercentage)) / 100))
	}
	if xdlJob.Spec.MinFinishWorkerNum != nil {
		return *xdlJob.Spec.MinFinishWorkerNum
	}
	// Min finish priority not set, all workers has to be finish successfully.
	return workersNum
}
