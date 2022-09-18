/*
Copyright 2022 The Alibaba Authors.

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

package torchelastic

import (
	"context"
	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	logger "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
)

func (ts *TorchElasticController) start(ctx context.Context, cancel context.CancelFunc, name, namespace string) {
	sharedPytorchJob := &training.PyTorchJob{}
	jobName := name
	jobNamespace := namespace

	// Create metrics for each torch elastic job.
	ts.locker.Lock()
	if _, ok := ts.metrics[jobName]; !ok {
		ts.metrics[jobName] = make(map[int32][]MetricObservation)
	}
	ts.locker.Unlock()

	err := ts.Client.Get(ctx, types.NamespacedName{Namespace: jobNamespace, Name: jobName}, sharedPytorchJob)
	if err != nil {
		logger.Infof("try to get job %s from namespace %s but it has been deleted", jobName, jobNamespace)
		// cancel the elastic scaling process context of the deleted job.
		defer cancel()
		return
	}

	pytorchJob := sharedPytorchJob.DeepCopy()
	if pytorchJob.Spec.ElasticPolicy.MaxReplicas == nil || pytorchJob.Spec.ElasticPolicy.MinReplicas == nil {
		logger.Infof("pytorch job %s does not configure the max or min replicas", pytorchJob.Name)
		defer cancel()
		delete(ts.torchElasticJobs, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		return
	}

	if pytorchJob.Status.ElasticStatus == nil {
		initializeElasticStatuses(pytorchJob, training.PyTorchReplicaTypeWorker)
		if err := ts.UpdateJobStatusInApiServer(pytorchJob, &pytorchJob.Status); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				log.Info("fail to update pytorch job")
			}
		}
		return
	}

	jobStatus := pytorchJob.Status.DeepCopy()
	oldStatus := jobStatus.DeepCopy()
	if pytorchJob.Status.CompletionTime != nil || pytorchJob.DeletionTimestamp != nil {
		logger.Infof("job %s has been completed or deleted and does not need to do elastic scaling", pytorchJob.Name)
		defer cancel()
		delete(ts.torchElasticJobs, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		delete(ts.metrics, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		return
	}

	currentReplicas := *pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas

	// Wait for all pods running and judge whether there exists pending or failed pods.
	hasPendingPod, hasFailedPod := ts.waitForAllPodsRunning(pytorchJob)

	// If job has pending pods and current replicas are more than min replicas, return to the last replicas.
	if hasPendingPod && currentReplicas > *pytorchJob.Spec.ElasticPolicy.MinReplicas {
		lastReplicas := jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas
		*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = lastReplicas
		// Return to the last replicas.
		if err := ts.Client.Update(ctx, pytorchJob); err != nil {
			log.Info("fail to update replicas of pytorch job")
		}

		updateElasticStatusForPendingJob(pytorchJob, lastReplicas, training.PyTorchReplicaTypeWorker)
		if err := ts.UpdateJobStatusInApiServer(pytorchJob, &pytorchJob.Status); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				log.Info("fail to update pytorch job")
			}
		}
		return

		// If job has pending pods and current replicas equals to the min replicas, cancel the elastic scaling process context.
	} else if (hasPendingPod && currentReplicas == *pytorchJob.Spec.ElasticPolicy.MinReplicas) || hasFailedPod {
		defer cancel()
		logger.Info("pods did not reach the running state at min replicas or job is failed, so the elastic scaling controller shutdown")
		delete(ts.torchElasticJobs, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		return
	}

	if !hasPendingPod && jobStatus.ElasticStatus != nil && jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue == false {
		// If job metrics have reached the max, restart stale pods.
		if jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].ElasticCondition == apiv1.ElasticMaxMetric {
			pods, err := ts.GetPodsForJob(pytorchJob)
			if err != nil {
				logger.Warnf("Get Pods For Job error %v", err)
			}
			// Restart stale torch elastic pods.
			complete := ts.restartStalePytorchPods(pods, pytorchJob)
			if !complete {
				logger.Info("restart pods does not complete")
				return
			}
			logger.Info("restart pods has completed")
			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].ElasticCondition = apiv1.ElasticStop
			if err = ts.UpdateJobStatusInApiServer(pytorchJob, jobStatus); err != nil {
				if errors.IsConflict(err) {
					// retry later when update operation violates with etcd concurrency control.
					logger.Info("fail to update pytorch job status")
					return
				}
			}
			return
			// If current replicas reach the defined max replicas or elastic condition is stopped, return directly.
		} else if jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].ElasticCondition == apiv1.ElasticStop || jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].ElasticCondition == apiv1.ElasticMaxReplica {
			log.Info("Pytorch job does not need to be scaled")
			return
		}
	}

	// Read training logs from pytorch pods and save the observation.
	observation, err := read(ts.client, jobNamespace, GetDefaultWorkerName(jobName))
	if err != nil {
		logger.Infof("fail to read training logs: %v", err)
		return
	}

	ts.locker.Lock()
	defer ts.locker.Unlock()

	// Create metrics for current replicas.
	if _, ok := ts.metrics[jobName][currentReplicas]; !ok {
		ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
	}
	ts.metrics[jobName][currentReplicas] = append(ts.metrics[jobName][currentReplicas], observation)
	currentLength := len(ts.metrics[jobName][currentReplicas])
	logger.Infof("Current metric length: %d", currentLength)

	// If current metrics have reached the metric count, judge the next scaling replicas.
	if currentLength >= ts.metricCount {
		if currentReplicas > *pytorchJob.Spec.ElasticPolicy.MinReplicas && currentReplicas <= *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
			lastReplicas := jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas

			if ts.IsSatisfyElasticContinue(jobName, currentReplicas, lastReplicas) {
				if currentReplicas == *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
					updateElasticStatusForMaxReplicaJob(pytorchJob, training.PyTorchReplicaTypeWorker)
					ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
				} else {
					newReplicas := computeNewReplicas(currentReplicas)
					*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = newReplicas
					if err := ts.Client.Update(ctx, pytorchJob); err != nil {
						log.Info("fail to update pytorch job")
					}

					updateElasticStatusForContinueJob(pytorchJob, currentReplicas, newReplicas, training.PyTorchReplicaTypeWorker)
					if _, ok := ts.metrics[jobName][newReplicas]; !ok {
						ts.metrics[jobName][newReplicas] = make([]MetricObservation, 0)
					}
				}

			} else {
				*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = lastReplicas
				if err := ts.Client.Update(ctx, pytorchJob); err != nil {
					log.Info("fail to update pytorch job")
				}

				updateElasticStatusForMaxMetricJob(pytorchJob, currentReplicas, lastReplicas, training.PyTorchReplicaTypeWorker)
				ts.metrics[jobName][lastReplicas] = make([]MetricObservation, 0)
				ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
			}

		} else if currentReplicas == *pytorchJob.Spec.ElasticPolicy.MinReplicas && currentReplicas < *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
			newReplicas := computeNewReplicas(currentReplicas)
			*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = newReplicas
			if err := ts.Client.Update(ctx, pytorchJob); err != nil {
				log.Info("fail to update pytorch job")
			}

			updateElasticStatusForContinueJob(pytorchJob, currentReplicas, newReplicas, training.PyTorchReplicaTypeWorker)
			if _, ok := ts.metrics[jobName][newReplicas]; !ok {
				ts.metrics[jobName][newReplicas] = make([]MetricObservation, 0)
			}

		} else if currentReplicas == *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
			updateElasticStatusForMaxReplicaJob(pytorchJob, training.PyTorchReplicaTypeWorker)
			if _, ok := ts.metrics[jobName][currentReplicas]; ok {
				ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
			}

		}
	}

	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, pytorchJob.Status) {
		if err = ts.UpdateJobStatusInApiServer(pytorchJob, &pytorchJob.Status); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				logger.Info("fail to update pytorch job status")
				return
			}
		}
	}

	return
}
