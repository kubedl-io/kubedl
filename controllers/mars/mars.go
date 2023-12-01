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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/quota"
)

const marsConfig = "MARS_CLUSTER_DETAIL"

// MarsConfig is a struct representing the distributed Mars config.
// see https://github.com/mars-project/mars/issues/1458 for details.
type MarsConfig struct {
	Cluster ClusterSpec `json:"cluster"`
	Task    TaskSpec    `json:"task"`
}

type ClusterSpec map[string][]string

type TaskSpec struct {
	Type      string          `json:"type"`
	Index     int             `json:"index"`
	Resources *workerResource `json:"resources,omitempty"`
}

type workerResource struct {
	// Number of computation processes on CPUs.
	CPU int64 `json:"cpu_procs"`
	// Limit of physical memory in megabytes.
	Memory int64 `json:"phy_mem"`
	// Reserved for future use of cuda devices.
	// CUDADevices []string `json:"cuda_devices,omitempty"`
}

func marsConfigInJson(marJob *v1alpha1.MarsJob, rtype, index string) (string, error) {
	i, err := strconv.ParseInt(index, 0, 32)
	if err != nil {
		return "", err
	}

	cluster, err := genClusterSpec(marJob)
	if err != nil {
		return "", err
	}

	task := TaskSpec{
		Type:  rtype,
		Index: int(i),
	}

	// Mars worker will perceive allocated resources and real-time usage inside worker
	// process and report to scheduler.
	if strings.EqualFold(rtype, string(v1alpha1.MarsReplicaTypeWorker)) {
		resources := quota.GetPodResourceRequest(&corev1.Pod{
			Spec: *marJob.Spec.MarsReplicaSpecs[v1alpha1.MarsReplicaTypeWorker].Template.Spec.DeepCopy(),
		})
		task.Resources = &workerResource{
			CPU:    resources.Cpu().Value(),
			Memory: resources.Memory().Value(),
		}
	}

	cfg, err := json.Marshal(&MarsConfig{
		Cluster: cluster,
		Task:    task,
	})
	if err != nil {
		return "", err
	}
	return string(cfg), nil
}

func genClusterSpec(marsJob *v1alpha1.MarsJob) (ClusterSpec, error) {
	cs := make(ClusterSpec)

	for rtype, spec := range marsJob.Spec.MarsReplicaSpecs {
		// Skip worker replicas since there is only one way dependency from worker to scheduler/webservice,
		// and number of workers may auto-scaled but environment can not be updated in runtime.
		if rtype == v1alpha1.MarsReplicaTypeWorker {
			continue
		}

		rt := strings.ToLower(string(rtype))
		endpoints := make([]string, 0, *spec.Replicas)

		port, err := job_controller.GetPortFromJob(marsJob.Spec.MarsReplicaSpecs, rtype, v1alpha1.MarsJobDefaultContainerName, v1alpha1.MarsJobDefaultPortName)
		if err != nil {
			return nil, err
		}
		for i := int32(0); i < *spec.Replicas; i++ {
			// Headless service assigned a DNS A record for a name of the form "my-svc.my-namespace.svc.cluster.local".
			// And the last part "svc.cluster.local" is called cluster domain
			// which maybe different between kubernetes clusters.
			hostname := commonutil.GenGeneralName(marsJob.Name, rt, fmt.Sprintf("%d", i))
			svcName := hostname + "." + marsJob.Namespace + ".svc"
			endpoint := fmt.Sprintf("%s:%d", svcName, port)
			endpoints = append(endpoints, endpoint)
		}
		cs[rt] = endpoints
	}
	return cs, nil
}

func injectSpillDirsByGivenPaths(paths []string, template *corev1.PodTemplateSpec, container *corev1.Container) {
	if len(paths) == 0 {
		return
	}
	var sizeLimit *resource.Quantity
	if !container.Resources.Requests.StorageEphemeral().IsZero() {
		q := resource.MustParse(strconv.Itoa(int(container.Resources.Requests.StorageEphemeral().Value()) / len(paths)))
		sizeLimit = &q
	}
	for idx, path := range paths {
		volumeName := "mars-empty-dir-" + strconv.Itoa(idx)
		volume := corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		if sizeLimit != nil {
			volume.VolumeSource.EmptyDir.SizeLimit = sizeLimit
		}
		// Check volume has not mounted yet inside container.
		existed := false
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == volumeName {
				existed = true
				break
			}
		}
		if existed {
			continue
		}
		template.Spec.Volumes = append(template.Spec.Volumes, volume)
		// Mount empty dir to target path.
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: path,
		})
	}
}
func computeCacheMemSize(memLimit int, memTuningPolicy *v1alpha1.MarsWorkerMemoryTuningPolicy) int {
	cacheSize := -1
	if memTuningPolicy.WorkerCacheSize != nil {
		cacheSize = int(memTuningPolicy.WorkerCacheSize.Value())
	} else if memTuningPolicy.WorkerCachePercentage != nil {
		percentage := int(*memTuningPolicy.WorkerCachePercentage)
		if percentage > 100 {
			percentage = 100
		}
		cacheSize = (memLimit * percentage) / 100
	}
	return cacheSize
}
func mountSharedCacheToPath(cacheSize int64, path string, template *corev1.PodTemplateSpec, container *corev1.Container) {
	if path == "" {
		return
	}
	const cacheVolumeName = "mars-shared-cache"
	cacheVolumeSource := &corev1.EmptyDirVolumeSource{
		Medium:    corev1.StorageMediumMemory,
		SizeLimit: resource.NewQuantity(cacheSize, resource.DecimalSI),
	}
	hasVolume := false
	for i := range template.Spec.Volumes {
		if v := template.Spec.Volumes[i]; v.Name == cacheVolumeName {
			hasVolume = true
			// Cache volume found in pod template, override it with specific cache settings
			// for data consistency.
			v.VolumeSource = corev1.VolumeSource{EmptyDir: cacheVolumeSource}
			break
		}
	}
	if !hasVolume {
		volume := corev1.Volume{
			Name:         cacheVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: cacheVolumeSource},
		}
		// Append a new memory-typed volume with given size and mount
		// to container.
		template.Spec.Volumes = append(template.Spec.Volumes, volume)
	}
	// Iterate volumes and find whether cache volume has mounted or not.
	for i := range container.VolumeMounts {
		if vm := container.VolumeMounts[i]; vm.Name == cacheVolumeName && vm.MountPath == path {
			return
		}
	}
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      cacheVolumeName,
		MountPath: path,
	})
}
