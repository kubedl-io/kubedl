/*
Copyright 2021 The Alibaba Authors.

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

package job_controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// SatisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by
// the controller manager.
func (jc *JobController) SatisfyExpectations(job metav1.Object, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec) bool {
	satisfied := true
	key, err := KeyFunc(job)
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to get the key of job(%s/%s)", job.GetNamespace(), job.GetName()))
		return false
	}

	for rtype := range specs {
		// Check the expectations of the pods.
		expectationPodsKey := GenExpectationPodsKey(key, string(rtype))
		satisfied = satisfied && jc.Expectations.SatisfiedExpectations(expectationPodsKey)
	}

	for rtype := range specs {
		// Check the expectations of the services, not all kinds of jobs create service for all
		// replicas, so return true when at least one service adds/dels observed.
		expectationServicesKey := GenExpectationServicesKey(key, string(rtype))
		satisfied = satisfied || jc.Expectations.SatisfiedExpectations(expectationServicesKey)
	}
	return satisfied
}

func (jc *JobController) DeleteExpectations(job metav1.Object, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec) {
	key, err := KeyFunc(job)
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to get the key of job(%s/%s)", job.GetNamespace(), job.GetName()))
		return
	}

	for rtype := range specs {
		expectationPodsKey := GenExpectationPodsKey(key, string(rtype))
		expectationServicesKey := GenExpectationServicesKey(key, string(rtype))
		jc.Expectations.DeleteExpectations(expectationPodsKey)
		jc.Expectations.DeleteExpectations(expectationServicesKey)
	}
}
