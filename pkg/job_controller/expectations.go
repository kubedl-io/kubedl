package job_controller

import (
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SatisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by
// the controller manager.
func (jc *JobController) SatisfyExpectations(job metav1.Object, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec) bool {
	satisfied := true
	key, err := KeyFunc(job)
	if err != nil {
		return false
	}
	for rtype := range specs {
		// Check the expectations of the pods.
		expectationPodsKey := GenExpectationPodsKey(key, string(rtype))
		satisfied = satisfied && jc.Expectations.SatisfiedExpectations(expectationPodsKey)

		// Check the expectations of the services.
		expectationServicesKey := GenExpectationServicesKey(key, string(rtype))
		satisfied = satisfied && jc.Expectations.SatisfiedExpectations(expectationServicesKey)
	}
	return satisfied
}
