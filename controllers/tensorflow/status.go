// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tensorflow provides a Kubernetes controller for a TFJob resource.
package tensorflow

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

// updateGeneralJobStatus updates the status of job with given replica specs and job status.
func (r *TFJobReconciler) updateGeneralJobStatus(tfJob *training.TFJob, replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec,
	jobStatus *v1.JobStatus, restart bool) error {
	log.Info("Updating status", "TFJob name", tfJob.Name)

	previousRestarting := commonutil.IsRestarting(*jobStatus)
	previousFailed := commonutil.IsFailed(*jobStatus)

	// Check whether worker 0 has completed。
	worker0Completed := false
	allWorkersSucceed := false
	workerRep, workerFound := replicaSpecs[training.TFReplicaTypeWorker]
	if workerFound {
		succeed := int32(0)
		if jobStatus.ReplicaStatuses[training.TFReplicaTypeWorker] != nil {
			succeed = jobStatus.ReplicaStatuses[training.TFReplicaTypeWorker].Succeeded
		}
		allWorkersSucceed = *workerRep.Replicas == succeed
	}

	// Get all pods for tfJob.
	pods, err := r.GetPodsForJob(tfJob)
	if err != nil {
		log.Error(err, "getPodsForTFJob error")
		return err
	}

	// Get all pods for workers.
	pods, err = r.ctrl.FilterPodsForReplicaType(pods, strings.ToLower(string(training.TFReplicaTypeWorker)))
	if err != nil {
		log.Error(err, "FilterPodsForReplicaType error")
		return err
	}

	for _, pod := range pods {
		index, err := strconv.Atoi(pod.Labels[v1.ReplicaIndexLabel])
		if err != nil {
			log.Error(err, "Error when strconv.Atoi")
			continue
		}
		if index == 0 {
			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == training.TFJobDefaultContainerName && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					break
				}
			}
			if exitCode == 0 && pod.Status.Phase == corev1.PodSucceeded {
				worker0Completed = true
			}
			break
		}
	}

	// Set job start time.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
	}

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
		// Expect to have `replicas - succeeded` pods alive.
		expected := replicas - status.Succeeded
		running := status.Active
		failed := status.Failed

		// If the TFJob contains Chief or Master spec, then we will update the status
		// according to the Chief/Master spec.
		if job_controller.ContainsReplicaType(replicaSpecs, training.TFReplicaTypeChief, training.TFReplicaTypeMaster, v1.JobReplicaTypeAIMaster) {
			if training.IsTFJobChieforMaster(rtype) {
				if running > 0 {
					msg := fmt.Sprintf("TFJob %s is running.", tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						log.Error(err, "append tfjob condition error")
						return err
					}
				}
				// Conditions for marking job as succeeded:
				// 1. chief or master exit successfully with success policy is none.
				// 2. if success policy is AllWorkers, then wait util all workers succeed.
				// 3. aimaster is enabled and it exits successfully.
				succeed := expected == 0
				if rtype != v1.JobReplicaTypeAIMaster && workerFound && tfJob.Spec.SuccessPolicy != nil && *tfJob.Spec.SuccessPolicy == v1.SuccessPolicyAllWorkers {
					succeed = succeed && allWorkersSucceed
				}
				if succeed {
					msg := fmt.Sprintf("TFJob %s successfully completed.", tfJob.Name)
					r.recorder.Event(tfJob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobSucceeded, commonutil.JobSucceededReason, msg)
					if err != nil {
						log.Error(err, "append tfjob condition error")
						return err
					}
					r.ctrl.Metrics.SuccessInc()
				}
			}
		} else {
			if rtype == training.TFReplicaTypeWorker {
				// All workers are succeeded or worker 0 completed, leave a succeeded condition.
				// Leave a succeeded condition for the following two cases:
				// 1. If default success policy is used and worker 0 has completed.
				// 2. If `SuccessPolicyAllWorkers` success policy is used and all workers are succeeded.
				if expected == 0 || (worker0Completed && *tfJob.Spec.SuccessPolicy != v1.SuccessPolicyAllWorkers) {
					msg := fmt.Sprintf("TFJob %s successfully completed.", tfJob.Name)
					r.recorder.Event(tfJob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobSucceeded, commonutil.JobSucceededReason, msg)
					if err != nil {
						log.Error(err, "append tfjob condition error")
						return err
					}
					r.ctrl.Metrics.SuccessInc()
				} else if running > 0 {
					// Some workers are still running, leave a running condition.
					msg := fmt.Sprintf("TFJob %s is running.", tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						log.Error(err, "append tfjob condition error")
						return err
					}
				}
			}
		}

		if failed > 0 {
			if restart && rtype != v1.JobReplicaTypeAIMaster {
				msg := fmt.Sprintf("TFJob %s is restarting because %d %s replica(s) failed.", tfJob.Name, failed, rtype)
				r.recorder.Event(tfJob, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					log.Error(err, "failed to update tensorflow job conditions")
					return err
				}
				if !previousRestarting {
					r.ctrl.Metrics.FailureInc()
					r.ctrl.Metrics.RestartInc()
				}
			} else {
				msg := fmt.Sprintf("TFJob %s is failed because %d %s replica(s) failed.", tfJob.Name, failed, rtype)
				r.recorder.Event(tfJob, corev1.EventTypeNormal, commonutil.JobFailedReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobFailed, commonutil.JobFailedReason, msg)
				if err != nil {
					log.Error(err, "failed to update tensorflow job conditions")
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
