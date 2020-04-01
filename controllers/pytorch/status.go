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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pytorchv1 "github.com/alibaba/kubedl/api/pytorch/v1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

// updateGeneralJobStatus updates the status of job with given replica specs and job status.
func (r *PytorchJobReconciler) updateGeneralJobStatus(pytorchJob *pytorchv1.PyTorchJob,
	replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	log.Info("Updating status", "PytorchJob name", pytorchJob.Name, "restart", restart)

	// Set job status start time since this job has acknowledged by controller.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
	}

	previousRestarting := commonutil.IsRestarting(*jobStatus)
	previousFailed := commonutil.IsFailed(*jobStatus)

	for rtype, spec := range replicaSpecs {
		replicas := *spec.Replicas
		// If rtype in replica status not found, there must be a mistyped/invalid rtype in job spec,
		// and it has not been reconciled in previous processes, discard it.
		status, ok := jobStatus.ReplicaStatuses[rtype]
		if !ok {
			log.Info("skipping invalid replica type", "rtype", rtype)
			continue
		}
		expected := replicas - status.Succeeded
		running := status.Active
		failed := status.Failed

		log.Info("Update pytorch job status", "PyTorchJob", pytorchJob.Name,
			"ReplicaType", rtype, "expected", expected, "running", running, "failed", failed)

		if ContainMasterSpec(pytorchJob) {
			if rtype == pytorchv1.PyTorchReplicaTypeMaster {
				if running > 0 {
					msg := fmt.Sprintf("PyTorchJob %s is running.", pytorchJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						log.Info("Append job condition", " error:", err)
						return err
					}
				}
				if expected == 0 {
					msg := fmt.Sprintf("PyTorchJob %s is successfully completed.", pytorchJob.Name)
					r.recorder.Event(pytorchJob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobSucceeded, commonutil.JobSucceededReason, msg)
					if err != nil {
						log.Info("Append job condition", "error:", err)
						return err
					}
					r.ctrl.Metrics.SuccessInc()
				}
			}
		} else {
			log.Info("Invalid config: Job must contain master replica spec")
			return errors.New("invalid config: Job must contain master replica spec")
		}

		if failed > 0 {
			if restart {
				msg := fmt.Sprintf("PyTorchJob %s is restarting because %d %s replica(s) failed.", pytorchJob.Name, failed, rtype)
				r.recorder.Event(pytorchJob, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					log.Info("Append job condition", "error:", err)
					return err
				}
				if !previousRestarting {
					r.ctrl.Metrics.FailureInc()
					r.ctrl.Metrics.RestartInc()
				}
			} else {
				msg := fmt.Sprintf("PyTorchJob %s is failed because %d %s replica(s) failed.", pytorchJob.Name, failed, rtype)
				r.recorder.Event(pytorchJob, corev1.EventTypeNormal, commonutil.JobFailedReason, msg)
				if jobStatus.CompletionTime == nil {
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
	return nil
}

func onOwnerCreateFunc(r reconcile.Reconciler) func(e event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		pytorchJob, ok := e.Meta.(*pytorchv1.PyTorchJob)
		if !ok {
			return true
		}
		reconciler, ok := r.(*PytorchJobReconciler)
		if !ok {
			return true
		}
		reconciler.scheme.Default(pytorchJob)
		msg := fmt.Sprintf("PytorchJob %s is created.", e.Meta.GetName())
		if err := commonutil.UpdateJobConditions(&pytorchJob.Status, v1.JobCreated, commonutil.JobCreatedReason, msg); err != nil {
			log.Error(err, "append job condition error")
			return false
		}
		reconciler.ctrl.Metrics.CreatedInc()
		return true
	}
}
