package quota

import (
	v1 "k8s.io/api/core/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/quota/v1"
)

// SumUpContainersResources sum up resources aggregated from containers list.
func SumUpContainersResources(containers []v1.Container) v1.ResourceRequirements {
	sum := v1.ResourceRequirements{
		Limits:   make(v1.ResourceList),
		Requests: make(v1.ResourceList),
	}
	for idx := range containers {
		container := &containers[idx]
		sum.Requests = quota.Add(sum.Requests, container.Resources.Requests)
		sum.Limits = quota.Add(sum.Limits, container.Resources.Limits)
	}
	return sum
}

// MaximumContainersResources iterate resources in containers list and compute
// a maximum one for each resource.
func MaximumContainersResources(containers []v1.Container) v1.ResourceRequirements {
	max := v1.ResourceRequirements{
		Limits:   make(v1.ResourceList),
		Requests: make(v1.ResourceList),
	}
	for idx := range containers {
		container := &containers[idx]
		max.Requests = quota.Max(max.Requests, container.Resources.Requests)
		max.Limits = quota.Max(max.Limits, container.Resources.Limits)
	}
	return max
}

func ComputePodResourceRequest(pod *v1.Pod) v1.ResourceList {
	result := v1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		result = quota.Add(result, container.Resources.Requests)
	}
	// take max_resource(sum_pod, any_init_container)
	for _, container := range pod.Spec.InitContainers {
		result = quota.Max(result, container.Resources.Requests)
	}
	// If Overhead is being utilized, add to the total requests for the pod
	if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(features.PodOverhead) {
		result = quota.Add(result, pod.Spec.Overhead)
	}
	return result
}
