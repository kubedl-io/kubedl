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

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

// updateGeneralJobStatus updates the status of job with given replica specs and job status.
func (r *PytorchJobReconciler) updateGeneralJobStatus(pytorchJob *training.PyTorchJob,
	replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	log.Info("Updating status", "PytorchJob name", pytorchJob.Name, "restart", restart)

	// Set job status start time since this job has acknowledged by controller.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
	}

	previousRestarting := commonutil.IsRestarting(*jobStatus)
	previousFailed := commonutil.IsFailed(*jobStatus)
	allWorkersSucceed := false
	workerRep, workerFound := replicaSpecs[training.PyTorchReplicaTypeWorker]
	if workerFound {
		succeed := int32(0)
		if jobStatus.ReplicaStatuses[training.PyTorchReplicaTypeWorker] != nil {
			succeed = jobStatus.ReplicaStatuses[training.PyTorchReplicaTypeWorker].Succeeded
		}
		allWorkersSucceed = *workerRep.Replicas == succeed
	}

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

		if job_controller.ContainsReplicaType(replicaSpecs, training.PyTorchReplicaTypeMaster, v1.JobReplicaTypeAIMaster) {
			if rtype == training.PyTorchReplicaTypeMaster || rtype == v1.JobReplicaTypeAIMaster {
				if running > 0 {
					msg := fmt.Sprintf("PyTorchJob %s is running.", pytorchJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						log.Info("Append job condition", " error:", err)
						return err
					}
				}
				// Conditions for marking job as succeeded:
				// 1. master exit successfully with success policy is none.
				// 2. if success policy is AllWorkers, then wait util all workers succeed.
				// 3. aimaster is enabled and it exits successfully.
				succeed := replicas > 0 && expected == 0
				if rtype != v1.JobReplicaTypeAIMaster && workerFound {
					succeed = succeed && allWorkersSucceed
				}
				if succeed {
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
			if restart && rtype != v1.JobReplicaTypeAIMaster {
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
