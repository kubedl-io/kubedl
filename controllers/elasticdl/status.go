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

package elasticdl

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	elasticdlv1alpha1 "github.com/alibaba/kubedl/api/elasticdljob/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

// updateGeneralJobStatus updates the status of job with given replica specs and job status.
func (r *ElasticDLJobReconciler) updateGeneralJobStatus(elasticdlJob *elasticdlv1alpha1.ElasticDLJob,
	replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	log.Info("Updating status", "ElasticDLJob name", elasticdlJob.Name, "restart", restart)

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

		log.Info("Update elasticdl job status", "ElasticDLJob", elasticdlJob.Name,
			"ReplicaType", rtype, "expected", expected, "running", running, "failed", failed)

		if ContainMasterSpec(elasticdlJob) {
			if rtype == elasticdlv1alpha1.ElasticDLReplicaTypeMaster {
				if running > 0 {
					msg := fmt.Sprintf("ElasticDLJob %s is running.", elasticdlJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, v1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						log.Info("Append job condition", " error:", err)
						return err
					}
				}
				if expected == 0 {
					msg := fmt.Sprintf("ElasticDLJob %s is successfully completed.", elasticdlJob.Name)
					r.recorder.Event(elasticdlJob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
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
				msg := fmt.Sprintf("ElasticDLJob %s is restarting because %d %s replica(s) failed.", elasticdlJob.Name, failed, rtype)
				r.recorder.Event(elasticdlJob, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
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
				msg := fmt.Sprintf("ElasticDLJob %s is failed because %d %s replica(s) failed.", elasticdlJob.Name, failed, rtype)
				r.recorder.Event(elasticdlJob, corev1.EventTypeNormal, commonutil.JobFailedReason, msg)
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
		elasticdlJob, ok := e.Meta.(*elasticdlv1alpha1.ElasticDLJob)
		if !ok {
			return true
		}
		reconciler, ok := r.(*ElasticDLJobReconciler)
		if !ok {
			return true
		}
		reconciler.scheme.Default(elasticdlJob)
		msg := fmt.Sprintf("ElasticDLJob %s is created.", e.Meta.GetName())
		if err := commonutil.UpdateJobConditions(&elasticdlJob.Status, v1.JobCreated, commonutil.JobCreatedReason, msg); err != nil {
			log.Error(err, "append job condition error")
			return false
		}
		reconciler.ctrl.Metrics.CreatedInc()
		return true
	}
}
