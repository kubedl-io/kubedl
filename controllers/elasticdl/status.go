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

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

// updateGeneralJobStatus updates the status of job with given replica specs and job status.
func (r *ElasticDLJobReconciler) updateGeneralJobStatus(elasticdlJob *training.ElasticDLJob,
	replicaSpecs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, restart bool) error {
	log.Info("Updating status", "ElasticDLJob name", elasticdlJob.Name, "restart", restart)

	// Set job status start time since this job has acknowledged by controller.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
	}

	previousRestarting := commonutil.IsRestarting(*jobStatus)
	previousFailed := commonutil.IsFailed(*jobStatus)

	// An ElasticDLJob only contains a master replica spec.
	rtype := training.ElasticDLReplicaTypeMaster
	if ContainMasterSpec(elasticdlJob) {
		replicas := *replicaSpecs[rtype].Replicas
		status := jobStatus.ReplicaStatuses[rtype]
		expected := replicas - status.Succeeded
		running := status.Active
		failed := status.Failed

		log.Info("Update elasticdl job status", "ElasticDLJob", elasticdlJob.Name,
			"ReplicaType", rtype, "expected", expected, "running", running, "failed", failed)

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
	} else {
		log.Info("Invalid config: Job must contain master replica spec")
		return errors.New("invalid config: Job must contain master replica spec")
	}
	return nil
}
