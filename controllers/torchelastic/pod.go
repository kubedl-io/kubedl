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
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/controllers/pytorch"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/concurrent"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	patchutil "github.com/alibaba/kubedl/pkg/util/patch"
	kruisev1alpha1 "github.com/openkruise/kruise/apis/apps/v1alpha1"
	logger "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"strconv"
	"strings"
)

func (ts *TorchElasticController) recreatePodContainers(job *trainingv1alpha1.PyTorchJob, pod *corev1.Pod, generation string) error {
	crr := kruisev1alpha1.ContainerRecreateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Labels: map[string]string{
				v1.LabelGeneration: generation,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "Pod",
					Name:               pod.Name,
					UID:                pod.UID,
					Controller:         pointer.BoolPtr(false),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
				{
					APIVersion:         job.APIVersion,
					Kind:               job.Kind,
					Name:               job.Name,
					UID:                job.UID,
					Controller:         pointer.BoolPtr(false),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
			},
		},
		Spec: kruisev1alpha1.ContainerRecreateRequestSpec{
			PodName:  pod.Name,
			Strategy: &kruisev1alpha1.ContainerRecreateRequestStrategy{OrderedRecreate: false},
		},
	}

	for ci := range pod.Spec.Containers {
		container := &pod.Spec.Containers[ci]
		crr.Spec.Containers = append(crr.Spec.Containers, kruisev1alpha1.ContainerRecreateRequestContainer{Name: container.Name})
	}
	return ts.Client.Create(context.Background(), &crr)
}

func (ts *TorchElasticController) restartStaleWorker(job *trainingv1alpha1.PyTorchJob, pod *corev1.Pod, worldSize, generation int64) (completed bool, err error) {
	expectedWorldSize := strconv.FormatInt(worldSize, 10)
	expectedGeneration := strconv.FormatInt(generation, 10)
	podKey := pod.Namespace + "/" + pod.Name

	if job.Annotations[pytorch.AnnotationReadyToRestartWorker] == "true" && !k8sutil.IsPodActive(pod) {
		err = ts.Client.Delete(context.Background(), pod)
		return err == nil, err
	}

	if pod.Labels[v1.LabelGeneration] == expectedGeneration {
		return true, nil
	}

	log.Info("refresh stale pod to latest generation", "pod", podKey, "generation", generation)

	completed, err = ts.restartWorkerInKruiseProtocol(job, pod, expectedWorldSize, expectedGeneration)
	if !completed {
		return false, err
	}

	// Finally, incremental generation for current worker and mark refreshment done.
	patch := patchutil.NewStrategicPatch()
	patch.InsertLabel(v1.LabelGeneration, expectedGeneration)
	err = ts.Client.Patch(context.Background(), pod, patch)
	if err != nil {
		return false, err
	}
	logger.Infof("succeed to refresh pod to generation: %v", generation)
	return true, nil
}

func (ts *TorchElasticController) restartWorkerInKruiseProtocol(job *trainingv1alpha1.PyTorchJob, pod *corev1.Pod, expectedWorldSize, expectedGeneration string) (completed bool, err error) {
	podKey := pod.Namespace + "/" + pod.Name
	crr := kruisev1alpha1.ContainerRecreateRequest{}
	if curWorldSize, ok := pod.Annotations[pytorch.AnnotationWorldSize]; !ok || curWorldSize != expectedWorldSize {
		log.Info("update latest world size of pytorch",
			"key", podKey, "current world size", curWorldSize, "target world size", expectedWorldSize)
		patch := patchutil.NewStrategicPatch()
		patch.InsertAnnotation(pytorch.AnnotationWorldSize, expectedWorldSize)
		if err = ts.Client.Patch(context.Background(), pod, patch); err != nil {
			log.Error(err, "failed to refresh world-size of stale worker", "pod", podKey, "world size", expectedWorldSize)
			return false, err
		}
		return false, nil
	}

	if err = ts.Client.Get(context.Background(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &crr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Not found ContainerRecreateRequest")
			return false, ts.recreatePodContainers(job, pod, expectedGeneration)
		}

		log.Error(err, "failed to get latest container-recreate-request for stale worker",
			"pod", podKey)
		return false, err
	}
	// crr created in previous round, clean it.
	if crr.Labels[v1.LabelGeneration] != expectedGeneration {
		if err = ts.Client.Delete(context.Background(), &crr); err != nil {
			return false, err
		}
		return false, ts.recreatePodContainers(job, pod, expectedGeneration)
	}

	if crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestFailed {
		logger.Infof("failed to restart containers of pod %s/%s, fallback to recreate pod", pod.Namespace, pod.Name)
		err = ts.Client.Delete(context.Background(), pod)
		return err == nil, err
	}

	recreateDone := crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestCompleted || crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestSucceeded
	if !recreateDone {
		logger.Error("container recreate request has not completed yet", "pod", podKey)
		return false, nil
	}

	// Finalize container-recreate-request object once it completes, because elastic scaling is repeatable
	// and 'crr' request will be re-initiated.
	defer ts.Client.Delete(context.Background(), &crr)

	logger.Info("ContainerRecreateSucceed", "succeed to recreate containers in stale worker: %s", podKey)
	return true, nil
}

func FilterRunningPods(pods []*corev1.Pod) []*corev1.Pod {
	var result []*corev1.Pod
	for _, p := range pods {
		if podRunning(p) {
			result = append(result, p)
		} else {
			deletionTimeStamp := "N/A"
			if p.DeletionTimestamp != nil {
				deletionTimeStamp = p.DeletionTimestamp.String()
			}
			logger.Infof("Ignoring inactive pod %v/%v in state %v, deletion time %s",
				p.Namespace, p.Name, p.Status.Phase, deletionTimeStamp)
		}
	}
	return result
}

func (ts *TorchElasticController) restartStalePytorchPods(pods []*corev1.Pod, pytorchJob *trainingv1alpha1.PyTorchJob) (completed bool) {

	runningPods := FilterRunningPods(pods)
	_, stalePods := k8sutil.FilterStalePodsByReplicaType(runningPods, pytorchJob.Generation, strings.ToLower(string(v1.JobReplicaTypeAIMaster)))
	staleWorkers := stalePods[strings.ToLower(string(trainingv1alpha1.PyTorchReplicaTypeWorker))]
	totalReplicas := len(stalePods)
	workerNums := len(staleWorkers)
	logger.Infof("worker nums: %d", workerNums)

	if pytorchJob.Annotations[pytorch.AnnotationReadyToRestartWorker] == "false" {
		log.Info("PytorchJob does not need to restart workers")
		return false
	}

	tickets := 100 // max semaphore tickets limited.
	if len(staleWorkers) < 100 {
		tickets = len(staleWorkers)
	}
	sema := concurrent.NewSemaphore(tickets)
	for _, pod := range staleWorkers {
		sema.Acquire()

		go func(worker *corev1.Pod) {
			defer sema.Release()
			if completed, err := ts.restartStaleWorker(pytorchJob, worker, int64(totalReplicas), pytorchJob.Generation); err != nil {
				logger.Warnf("Restart worker %s failed becasue error %v", worker.Name, err)
			} else if completed {
				workerNums--
			}
		}(pod)
	}
	// block until all semaphore is released.
	sema.Wait()
	if workerNums != 0 {
		log.Info("refresh stale workers has not completed yet", "key", pytorchJob.Namespace+"/"+pytorchJob.Name)
		return false
	}

	if len(stalePods) == 0 || workerNums == 0 {
		log.Info("all pods are in latest generation, mark ready-to-start-worker as false")
		patch := patchutil.NewMergePatch()
		patch.InsertAnnotation(pytorch.AnnotationReadyToRestartWorker, "false")

		if err := ts.Client.Patch(context.Background(), pytorchJob, patch); err != nil {
			logger.Infof("fail to patch pytorchJob: %v", err)
			return false
		}
		logger.Infof("pytorch job %s/%s elastic scaling successfully finished, total replicas: %v", pytorchJob.Namespace, pytorchJob.Name, totalReplicas)
		completed = true
	}

	return completed

}

func (ts *TorchElasticController) waitForAllPodsRunning(pytorchJob *trainingv1alpha1.PyTorchJob) (hasPendingPod, hasFailedPod bool) {
	pods, err := ts.GetPodsForJob(pytorchJob)
	if err != nil {
		logger.Warnf("Get Pods For Job error %v", err)
	}

	// Wait for all pods running with timeout seconds.
	waitErr := wait.PollImmediate(interval, podReadyTimeout, func() (bool, error) {
		for _, pod := range pods {
			if isRunning := podRunning(pod); !isRunning {
				return false, nil
			}
		}
		return true, nil
	})

	if waitErr != nil {
		logger.Info("pods did not reach the running state")
	}

	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodPending {
			hasPendingPod = true
			break
		}
	}
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodFailed {
			hasFailedPod = true
			break
		}
	}
	return
}
