package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

var SchemeGroupVersion = GroupVersion

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	SchemeBuilder.SchemeBuilder.Register(addDefaultingFuncs)
}

// Resource is required by pkg/client/listers/...
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func cleanPodPolicyPointer(cleanPodPolicy v1.CleanPodPolicy) *v1.CleanPodPolicy {
	c := cleanPodPolicy
	return &c
}

func enableFallbackToLogsOnErrorTerminationMessagePolicy(podSpec *corev1.PodSpec) {
	for ci := range podSpec.Containers {
		c := &podSpec.Containers[ci]
		if c.TerminationMessagePolicy == "" {
			c.TerminationMessagePolicy = corev1.TerminationMessageFallbackToLogsOnError
		}
	}
}

func ExtractMetaFieldsFromObject(obj client.Object) (replicas map[v1.ReplicaType]*v1.ReplicaSpec, status *v1.JobStatus, schedPolicy *v1.SchedulingPolicy) {
	switch typed := obj.(type) {
	case *TFJob:
		return typed.Spec.TFReplicaSpecs, &typed.Status, typed.Spec.SchedulingPolicy
	case *PyTorchJob:
		return typed.Spec.PyTorchReplicaSpecs, &typed.Status, typed.Spec.SchedulingPolicy
	case *XDLJob:
		return typed.Spec.XDLReplicaSpecs, &typed.Status, typed.Spec.SchedulingPolicy
	case *MPIJob:
		return typed.Spec.MPIReplicaSpecs, &typed.Status, typed.Spec.SchedulingPolicy
	case *XGBoostJob:
		return typed.Spec.XGBReplicaSpecs, &typed.Status.JobStatus, typed.Spec.RunPolicy.SchedulingPolicy
	case *MarsJob:
		return typed.Spec.MarsReplicaSpecs, &typed.Status.JobStatus, typed.Spec.SchedulingPolicy
	case *ElasticDLJob:
		return typed.Spec.ElasticDLReplicaSpecs, &typed.Status, typed.Spec.SchedulingPolicy
	}
	return nil, nil, nil
}
