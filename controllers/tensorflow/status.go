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

	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc(r reconcile.Reconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		tfJob, ok := e.Meta.(*tfv1.TFJob)
		if !ok {
			return true
		}
		reconciler, ok := r.(*TFJobReconciler)
		if !ok {
			return true
		}
		reconciler.scheme.Default(tfJob)
		msg := fmt.Sprintf("TFJob %s is created.", e.Meta.GetName())
		if err := commonutil.UpdateJobConditions(&tfJob.Status, v1.JobCreated, commonutil.JobCreatedReason, msg); err != nil {
			log.Error(err, "append job condition error")
			return false
		}
		log.Info(msg)
		if reconciler.ctrl.MetricsCounter != nil {
			reconciler.ctrl.MetricsCounter.CreatedInc()
			reconciler.ctrl.MetricsCounter.PendingInc()
		}
		return true
	}
}

// updateGeneralJobStatus updates the status of job with given replica specs and job status.
func (r *TFJobReconciler) updateGeneralJobStatus(tfJob *tfv1.TFJob, replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus) error {
	log.Info("Updating status", "TFJob name", tfJob.Name)

	// Check whether worker 0 has completed。
	worker0Completed := false

	// Get all pods for tfJob.
	pods, err := r.GetPodsForJob(tfJob)
	if err != nil {
		log.Error(err, "getPodsForTFJob error")
		return err
	}

	// Get all pods for workers.
	pods, err = r.ctrl.FilterPodsForReplicaType(pods, strings.ToLower(string(tfv1.TFReplicaTypeWorker)))
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
				if status.Name == tfv1.DefaultContainerName && state.Terminated != nil {
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

	// If job is previous running, when it transfers to state failed or restarting,
	// running metrics will do dec.
	previousRunning := commonutil.IsRunning(*jobStatus)

	// Iterate all replicas for counting and try to update start time.
	for rtype, spec := range replicaSpecs {

		replicas := *spec.Replicas
		// Expect to have `replicas - succeeded` pods alive.
		expected := replicas - jobStatus.ReplicaStatuses[rtype].Succeeded
		running := jobStatus.ReplicaStatuses[rtype].Active
		failed := jobStatus.ReplicaStatuses[rtype].Failed

		// If the TFJob contains Chief or Master spec, then we will update the status
		// according to the Chief/Master spec.
		if ContainChieforMasterSpec(tfJob) {
			if tfv1.IsChieforMaster(rtype) {
				if running > 0 {
					msg := fmt.Sprintf("TFJob %s is running.", tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						log.Error(err, "append tfjob condition error")
						return err
					}
					if r.ctrl.MetricsCounter != nil && !previousRunning {
						r.ctrl.MetricsCounter.RunningInc()
						r.ctrl.MetricsCounter.PendingDec()
						r.ctrl.MetricsHistogram.LaunchDelay(tfJob, *jobStatus)
					}
				}
				if expected == 0 {
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
					if r.ctrl.MetricsCounter != nil {
						r.ctrl.MetricsCounter.SuccessInc()
						r.ctrl.MetricsCounter.RunningDec()
					}
				}
			}
		} else {
			if rtype == tfv1.TFReplicaTypeWorker {
				// All workers are succeeded or worker 0 completed, leave a succeeded condition.
				if expected == 0 || worker0Completed {
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
					if r.ctrl.MetricsCounter != nil {
						r.ctrl.MetricsCounter.SuccessInc()
						r.ctrl.MetricsCounter.RunningDec()
					}
				} else if running > 0 {
					// Some workers are still running, leave a running condition.
					msg := fmt.Sprintf("TFJob %s is running.", tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						log.Error(err, "append tfjob condition error")
						return err
					}
					if r.ctrl.MetricsCounter != nil && !previousRunning {
						r.ctrl.MetricsCounter.RunningInc()
						r.ctrl.MetricsCounter.PendingDec()
						r.ctrl.MetricsHistogram.LaunchDelay(tfJob, *jobStatus)
					}
				}
			}
		}

		if failed > 0 {
			if spec.RestartPolicy == v1.RestartPolicyExitCode {
				msg := fmt.Sprintf("TFJob %s is restarting because %d %s replica(s) failed.", tfJob.Name, failed, rtype)
				r.recorder.Event(tfJob, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, v1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					log.Error(err, "failed to update tensorflow job conditions")
					return err
				}
				if r.ctrl.MetricsCounter != nil {
					r.ctrl.MetricsCounter.FailureInc()
					r.ctrl.MetricsCounter.RestartInc()
					if previousRunning {
						r.ctrl.MetricsCounter.RunningDec()
					}
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
				if r.ctrl.MetricsCounter != nil {
					r.ctrl.MetricsCounter.FailureInc()
					if previousRunning {
						r.ctrl.MetricsCounter.RunningDec()
					}
				}
			}
		}
	}
	return nil
}
