package resource_utils

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	quotav1 "k8s.io/apiserver/pkg/quota/v1"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// SumUpContainersResources sum up resources aggregated from containers list.
func SumUpContainersResources(containers []v1.Container) v1.ResourceRequirements {
	sum := v1.ResourceRequirements{
		Limits:   make(v1.ResourceList),
		Requests: make(v1.ResourceList),
	}
	for idx := range containers {
		container := &containers[idx]
		sum.Requests = quotav1.Add(sum.Requests, container.Resources.Requests)
		sum.Limits = quotav1.Add(sum.Limits, container.Resources.Limits)
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
		max.Requests = quotav1.Max(max.Requests, container.Resources.Requests)
		max.Limits = quotav1.Max(max.Limits, container.Resources.Limits)
	}
	return max
}

func JobResourceRequests(replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) (normal, spot v1.ResourceList) {
	for _, rspec := range replicas {
		resources := ComputePodSpecResourceRequest(&rspec.Template.Spec)
		replicas := int32(1)
		if rspec.Replicas != nil {
			replicas = *rspec.Replicas
		}
		if rspec.SpotReplicaSpec != nil && rspec.SpotReplicaSpec.SpotReplicaNumber > 0 {
			replicas -= rspec.SpotReplicaSpec.SpotReplicaNumber
			spotDelta := Multiply(int64(rspec.SpotReplicaSpec.SpotReplicaNumber), resources)
			spot = quotav1.Add(spot, spotDelta)
			if replicas < 0 {
				replicas = 0
			}
		}
		resources = Multiply(int64(replicas), resources)
		normal = quotav1.Add(normal, resources)
	}

	return normal, spot
}

func ReplicaResourceRequests(rspec *apiv1.ReplicaSpec) v1.ResourceList {
	resources := ComputePodSpecResourceRequest(&rspec.Template.Spec)
	replicas := int32(1)
	if rspec.Replicas != nil {
		replicas = *rspec.Replicas
	}
	return Multiply(int64(replicas), resources)
}

// ComputePodResourceRequest returns the requested resource of the Pod
func ComputePodResourceRequest(pod *v1.Pod) v1.ResourceList {
	return ComputePodSpecResourceRequest(&pod.Spec)
}

// ComputePodSpecResourceRequest returns the requested resource of the PodSpec
func ComputePodSpecResourceRequest(spec *v1.PodSpec) v1.ResourceList {
	result := v1.ResourceList{}
	for _, container := range spec.Containers {
		result = quotav1.Add(result, container.Resources.Requests)
	}
	// take max_resource(sum_pod, any_init_container)
	for _, container := range spec.InitContainers {
		result = quotav1.Max(result, container.Resources.Requests)
	}
	// If Overhead is being utilized, add to the total requests for the pod
	if spec.Overhead != nil {
		result = quotav1.Add(result, spec.Overhead)
	}
	return result
}

func Min(a, b resource.Quantity) resource.Quantity {
	if a.Cmp(b) < 0 {
		return a.DeepCopy()
	}
	return b.DeepCopy()
}

func PodRequestsForGPU(pod *v1.Pod) bool {
	for idx := range pod.Spec.Containers {
		c := &pod.Spec.Containers[idx]
		if containsResourceGPU(c.Resources.Requests) {
			return true
		}
	}
	return false
}

func JobRequestsForGPU(specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec) bool {
	for _, spec := range specs {
		if ReplicaRequestsForGPU(spec) {
			return true
		}
	}
	return false
}

func ReplicaRequestsForGPU(spec *apiv1.ReplicaSpec) bool {
	for idx := range spec.Template.Spec.Containers {
		c := &spec.Template.Spec.Containers[idx]
		if containsResourceGPU(c.Resources.Requests) || containsResourceGPU(c.Resources.Limits) {
			return true
		}
	}
	return false
}

func containsResourceGPU(req v1.ResourceList) bool {
	if req == nil {
		return false
	}
	_, ok := req[apiv1.ResourceNvidiaGPU]
	return ok
}

// GetGpuResource get gpu from resource list if gpu resource exists
func GetGpuResource(resourceList v1.ResourceList) *resource.Quantity {
	if val, ok := resourceList[apiv1.ResourceNvidiaGPU]; ok {
		return &val
	}
	return &resource.Quantity{Format: resource.DecimalSI}
}

// Multiply multiplies resources with given factor for each named resource.
func Multiply(factor int64, res v1.ResourceList) v1.ResourceList {
	result := v1.ResourceList{}
	for key, value := range res {
		scaled := value
		scaled.Set(factor * scaled.Value())
		result[key] = scaled
	}
	return result
}

// AnyLessThanOrEqual returns true if a < b for any key in b
// If false, it returns the keys in a that exceeded b
func AnyLessThanOrEqual(a v1.ResourceList, b v1.ResourceList) (bool, []v1.ResourceName) {
	var (
		resourceNames []v1.ResourceName
		result        = false
	)
	for key, value := range b {
		if other, found := a[key]; found {
			if other.Cmp(value) < 0 {
				result = true
				resourceNames = append(resourceNames, key)
			}
		}
	}
	return result, resourceNames
}
