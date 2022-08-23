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
	"fmt"
	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func makeElasticJobName(name, namespace string) string {
	return name + "-" + namespace
}

// UpdateJobStatusInApiServer updates the job status in API server
func (ts *TorchElasticController) UpdateJobStatusInApiServer(job interface{}, jobStatus *apiv1.JobStatus) error {
	torchElasticJob, ok := job.(*training.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PytorchJob", torchElasticJob)
	}
	var jobCpy *training.PyTorchJob
	// Job status passed in differs with status in job, update in basis of the passed in one.
	jobCpy = torchElasticJob.DeepCopy()
	jobCpy.Status = *jobStatus.DeepCopy()
	return ts.Status().Update(context.Background(), jobCpy)
}

// initializeReplicaStatuses initializes the ElasticStatuses for replica.
func initializeElasticStatuses(pytorchJob *training.PyTorchJob, rtype apiv1.ReplicaType) {
	jobStatus := &pytorchJob.Status
	if jobStatus.ElasticStatus == nil {
		jobStatus.ElasticStatus = make(map[apiv1.ReplicaType]*apiv1.ElasticScalingStatus)
	}

	jobStatus.ElasticStatus[rtype] = &apiv1.ElasticScalingStatus{ElasticCondition: apiv1.ElasticStart}
	jobStatus.ElasticStatus[rtype].CurrentReplicas = *pytorchJob.Spec.PyTorchReplicaSpecs[rtype].Replicas
	jobStatus.ElasticStatus[rtype].Continue = true
	now := metav1.Now()
	jobStatus.ElasticStatus[rtype].LastUpdateTime = &now
}

func updateElasticStatusForPendingJob(pytorchJob *training.PyTorchJob, lastReplicas int32, rtype apiv1.ReplicaType) {
	jobStatus := &pytorchJob.Status
	jobStatus.ElasticStatus[rtype].Continue = false
	jobStatus.ElasticStatus[rtype].LastReplicas = jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].CurrentReplicas
	jobStatus.ElasticStatus[rtype].CurrentReplicas = lastReplicas
	jobStatus.ElasticStatus[rtype].Message = "There exists pending pods, return to the last replicas"
	now := metav1.Now()
	jobStatus.ElasticStatus[rtype].LastUpdateTime = &now
	jobStatus.ElasticStatus[rtype].ElasticCondition = apiv1.ElasticStop
}

func updateElasticStatusForContinueJob(pytorchJob *training.PyTorchJob, currentReplicas, newReplicas int32, rtype apiv1.ReplicaType) {
	jobStatus := &pytorchJob.Status
	jobStatus.ElasticStatus[rtype].LastReplicas = currentReplicas
	jobStatus.ElasticStatus[rtype].CurrentReplicas = newReplicas
	jobStatus.ElasticStatus[rtype].Message = "Pytorch job continues to be scaled"
	now := metav1.Now()
	jobStatus.ElasticStatus[rtype].LastUpdateTime = &now
	jobStatus.ElasticStatus[rtype].Continue = true
	jobStatus.ElasticStatus[rtype].ElasticCondition = apiv1.ElasticContinue
}

func updateElasticStatusForMaxReplicaJob(pytorchJob *training.PyTorchJob, rtype apiv1.ReplicaType) {
	jobStatus := &pytorchJob.Status
	jobStatus.ElasticStatus[rtype].Message = "Pytorch job has reached the max replicas"
	jobStatus.ElasticStatus[rtype].Continue = false
	jobStatus.ElasticStatus[rtype].ElasticCondition = apiv1.ElasticMaxReplica
}

func updateElasticStatusForMaxMetricJob(pytorchJob *training.PyTorchJob, currentReplicas, lastReplicas int32, rtype apiv1.ReplicaType) {
	jobStatus := &pytorchJob.Status
	jobStatus.ElasticStatus[rtype].CurrentReplicas = lastReplicas
	jobStatus.ElasticStatus[rtype].LastReplicas = currentReplicas
	jobStatus.ElasticStatus[rtype].Message = "Pytorch job has reached the max metrics"
	now := metav1.Now()
	jobStatus.ElasticStatus[rtype].LastUpdateTime = &now
	jobStatus.ElasticStatus[rtype].Continue = false
	jobStatus.ElasticStatus[rtype].ElasticCondition = apiv1.ElasticMaxMetric
}

func (ts *TorchElasticController) IsSatisfyElasticContinue(jobName string, currentReplicas, lastReplicas int32) bool {
	currentLength := ts.metricCount
	currentLatency := ts.metrics[jobName][currentReplicas][currentLength-1].Latency
	lastReplicaLatency := ts.metrics[jobName][lastReplicas][currentLength-1].Latency
	//Decide whether the elastic scaling can continue by the ratio of batch training latency and replicas.
	return lastReplicaLatency/float64(lastReplicas) > currentLatency/float64(currentReplicas)
}

func computeNewReplicas(currentReplicas int32) int32 {
	// Double the replicas in the next elastic scaling loop.
	return currentReplicas * 2
}

func (ts *TorchElasticController) GetPodsForJob(job *training.PyTorchJob) ([]*v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: ts.GenLabels(job.Name),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podList := &v1.PodList{}
	err = ts.Client.List(context.Background(), podList, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	return commonutil.ToPodPointerList(podList.Items), nil
}
