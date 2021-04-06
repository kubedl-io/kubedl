package v1alpha1

import (
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
