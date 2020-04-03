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

package converters

import (
	"encoding/json"
	"errors"
	"fmt"
	"k8s.io/klog"
	"time"

	v1 "k8s.io/api/core/v1"
	quotav1 "k8s.io/kubernetes/pkg/quota/v1"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	"github.com/alibaba/kubedl/pkg/util/quota"
)

var (
	ErrNoDependentOwner   = errors.New("object has no dependent owner")
	ErrNoReplicaTypeLabel = fmt.Errorf("object has no replica type label [%s]", apiv1.ReplicaTypeLabel)
)

// ConvertPodToDMOPod converts a native pod object to dmo pod.
func ConvertPodToDMOPod(pod *v1.Pod, defaultContainerName string, region string) (*dmo.Pod, error) {
	klog.V(5).Infof("[ConvertPodToDMOPod] pod: %s/%s", pod.Namespace, pod.Name)
	dmoPod := &dmo.Pod{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		PodID:      string(pod.UID),
		Version:    pod.ResourceVersion,
		GmtCreated: pod.CreationTimestamp.Time,
	}

	if region != "" {
		dmoPod.DeployRegion = &region
	}

	id, _ := k8sutil.ResolveDependentOwner(pod)
	if id == "" {
		return nil, ErrNoDependentOwner
	}
	dmoPod.JobID = id

	rtype, ok := k8sutil.GetReplicaType(pod)
	if !ok {
		return nil, ErrNoReplicaTypeLabel
	}
	dmoPod.ReplicaType = rtype

	resources := computePodResources(&pod.Spec)
	resourcesBytes, err := json.Marshal(&resources)
	if err != nil {
		return nil, err
	}
	dmoPod.Resources = string(resourcesBytes)

	dmoPod.Deleted = util.IntPtr(0)
	dmoPod.IsInEtcd = util.IntPtr(1)
	if pod.Status.PodIP != "" {
		dmoPod.PodIP = &pod.Status.PodIP
	}
	if pod.Status.HostIP != "" {
		dmoPod.HostIP = &pod.Status.HostIP
	}

	if len(pod.Spec.Containers) == 0 {
		return dmoPod, nil
	}

	image := pod.Spec.Containers[0].Image
	for idx := 1; idx < len(pod.Spec.Containers); idx++ {
		if pod.Spec.Containers[idx].Name == defaultContainerName {
			image = pod.Spec.Containers[idx].Image
			break
		}
	}
	dmoPod.Image = image

	// Pod status Unknown defaulted.
	dmoPod.Status = v1.PodUnknown
	if len(pod.Status.ContainerStatuses) == 0 {
		return dmoPod, nil
	}

	containerStatus := pod.Status.ContainerStatuses[0]
	for idx := 1; idx < len(pod.Status.ContainerStatuses); idx++ {
		if pod.Status.ContainerStatuses[idx].Name == defaultContainerName {
			containerStatus = pod.Status.ContainerStatuses[idx]
			break
		}
	}
	dmoPod.Status = pod.Status.Phase

	switch pod.Status.Phase {
	case v1.PodPending:
		// Do nothing.
	case v1.PodRunning:
		if containerStatus.State.Running != nil {
			startedAt := containerStatus.State.Running.StartedAt
			dmoPod.GmtStarted = &startedAt.Time
		}
		if dmoPod.GmtStarted == nil || dmoPod.GmtStarted.IsZero() {
			dmoPod.GmtStarted = util.TimePtr(pod.CreationTimestamp.Time)
		}
	case v1.PodSucceeded, v1.PodFailed:
		if containerStatus.State.Terminated != nil {
			startedAt := containerStatus.State.Terminated.StartedAt
			finishedAt := containerStatus.State.Terminated.FinishedAt
			dmoPod.GmtStarted = &startedAt.Time
			dmoPod.GmtFinished = &finishedAt.Time
			if dmoPod.Status == v1.PodFailed {
				remark := fmt.Sprintf("Reason: %v\nExitCode: %v\nMessage: %v",
					containerStatus.State.Terminated.Reason,
					containerStatus.State.Terminated.ExitCode,
					containerStatus.State.Terminated.Message)
				dmoPod.Remark = &remark
			}
		}
		if dmoPod.GmtStarted == nil || dmoPod.GmtStarted.IsZero() {
			dmoPod.GmtStarted = util.TimePtr(pod.CreationTimestamp.Time)
		}
		if dmoPod.GmtFinished == nil || dmoPod.GmtFinished.IsZero() {
			dmoPod.GmtFinished = util.TimePtr(time.Now())
		}
	}

	return dmoPod, nil
}

func computePodResources(podSpec *v1.PodSpec) (resources v1.ResourceRequirements) {
	initResources := quota.MaximumContainersResources(podSpec.InitContainers)
	runtimeResources := quota.SumUpContainersResources(podSpec.Containers)
	resources.Requests = quotav1.Max(initResources.Requests, runtimeResources.Requests)
	resources.Limits = quotav1.Max(initResources.Limits, runtimeResources.Limits)
	return resources
}
