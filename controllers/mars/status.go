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
	"fmt"

	"github.com/alibaba/kubedl/api/marsjob/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	marsJobRestarting = "MarsJobRestarting"
	marsJobFailed     = "MarsJobFailed"
)

func (r *MarsJobReconciler) updateGeneralJobStatus(marsJob *v1alpha1.MarsJob, replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	log.Info("Updating status", "MarsJob name", marsJob.Name, "restart", restart)

	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
	}

	previousRestarting := commonutil.IsRestarting(*jobStatus)
	previousFailed := commonutil.IsFailed(*jobStatus)
	runningWorkers := int32(0)

	for rtype, spec := range replicaSpecs {
		// If rtype in replica status not found, there must be a mistyped/invalid rtype in job spec,
		// and it has not been reconciled in previous processes, discard it.
		status, ok := jobStatus.ReplicaStatuses[rtype]
		if !ok {
			log.Info("skipping invalid replica type", "rtype", rtype)
			continue
		}
		replicas := *spec.Replicas
		running := status.Active
		failed := status.Failed
		if rtype == v1alpha1.MarsReplicaTypeWorker {
			runningWorkers = running
		}

		log.Info("Update mars job status", "MarsJob", marsJob.Name,
			"ReplicaType", rtype, "running", running, "failed", failed)

		if failed > 0 {
			// Scheduler in mars job plays a role of scheduling operands and chunks, and it will
			// store intermediate states in memory, so the whole job failed once scheduler exits
			// unexpectedly.
			// TODO(SimonCqk): restart scheduler when it supports failover.
			if rtype == v1alpha1.MarsReplicaTypeScheduler {
				msg := fmt.Sprintf("MarsJob %s is failed because %d %s replica(s) failed", marsJob.Name, failed, rtype)
				r.recorder.Eventf(marsJob, corev1.EventTypeNormal, marsJobFailed, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				if err := commonutil.UpdateJobConditions(jobStatus, v1.JobFailed, marsJobFailed, msg); err != nil {
					log.Error(err, "failed to update mars job conditions")
					return err
				}
				if !previousFailed {
					r.ctrl.Metrics.FailureInc()
				}
			} else if restart {
				msg := fmt.Sprintf("MarsJob %s is restarting because %d %s replica(s) failed", marsJob.Name, failed, rtype)
				r.recorder.Eventf(marsJob, corev1.EventTypeWarning, marsJobRestarting, msg)
				if err := commonutil.UpdateJobConditions(jobStatus, v1.JobRestarting, marsJobRestarting, msg); err != nil {
					log.Error(err, "failed to update mars job conditions")
					return err
				}
				if !previousRestarting {
					r.ctrl.Metrics.FailureInc()
					r.ctrl.Metrics.RestartInc()
				}
			}
			return nil
		}

		// Mars job succeeds only when all schedulers exit expectedly.
		if rtype == v1alpha1.MarsReplicaTypeScheduler && status.Succeeded == replicas {
			msg := fmt.Sprintf("MarsJob %s has successfully completed.", marsJob.Name)
			r.recorder.Eventf(marsJob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
			if jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}
			if err := commonutil.UpdateJobConditions(jobStatus, v1.JobSucceeded, commonutil.JobSucceededReason, msg); err != nil {
				log.Error(err, "failed to update mars job conditions")
				return err
			}
			r.ctrl.Metrics.SuccessInc()
			return nil
		}
	}

	// Otherwise when workers start to run, leave it a running state, wait for all DAG-based tasks
	// complete and notify schedulers to exit.
	if runningWorkers > 0 {
		msg := fmt.Sprintf("MarsJob %s is running.", marsJob.Name)
		if err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg); err != nil {
			log.Error(err, "failed to update mars job conditions")
			return err
		}
	}
	return nil
}

func onOwnerCreateFunc(r reconcile.Reconciler) func(e event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		marsJob, ok := e.Meta.(*v1alpha1.MarsJob)
		if !ok {
			return false
		}
		reconciler, ok := r.(*MarsJobReconciler)
		if !ok {
			return false
		}
		reconciler.scheme.Default(marsJob)

		msg := fmt.Sprintf("MarsJob %s is created.", e.Meta.GetName())
		if err := commonutil.UpdateJobConditions(&marsJob.Status, v1.JobCreated, commonutil.JobCreatedReason, msg); err != nil {
			log.Error(err, "append job condition error")
			return false
		}
		reconciler.ctrl.Metrics.CreatedInc()
		return true
	}
}
