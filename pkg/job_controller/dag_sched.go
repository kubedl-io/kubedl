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
	"strings"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func (jc *JobController) dagConditionsReady(job metav1.Object, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, pods []*v1.Pod, dagConditions []apiv1.DAGCondition) bool {
	if len(dagConditions) == 0 {
		// No upstream vertex conditions, just continue.
		return true
	}
	klog.Infof("start to check DAG conditions of job %s/%s.", job.GetNamespace(), job.GetName())
	replicaPods := jc.SortPodsByReplicaType(pods, ReplicaTypes(specs))
	// Check all DAG conditions are ready.
	for i := range dagConditions {
		if !jc.upstreamReplicasReady(replicaPods, specs, dagConditions[i]) {
			klog.Infof("DAG condition has not ready, upstream: %s, on phase: %s",
				dagConditions[i].Upstream, dagConditions[i].OnPhase)
			return false
		}
	}
	klog.Infof("DAG conditions of job %s/%s has all ready.", job.GetNamespace(), job.GetName())
	return true
}
func (jc *JobController) upstreamReplicasReady(replicaPods map[apiv1.ReplicaType][]*v1.Pod, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, dagCondition apiv1.DAGCondition) bool {
	spec, ok := specs[dagCondition.Upstream]
	if !ok {
		// Upstream replica not exists in specs, treat it as a ready vertex.
		return true
	}
	pods := replicaPods[dagCondition.Upstream]
	replicas := *spec.Replicas
	// Check all pods has been created and reach expected replicas.
	if len(pods) < int(replicas) {
		klog.V(3).Infof("upstream pods has not reach expected replicas, expected: %d, now: %d", replicas, len(pods))
		return false
	}
	// Iterate all pods and check pod has stepped into expected phase.
	for _, pod := range pods {
		// Compares two pod phase and break when pod phase is behind dag-required-phase.
		if phaseComparator(pod.Status.Phase, dagCondition.OnPhase) < 0 {
			return false
		}
	}
	return true
}
func (jc *JobController) SortPodsByReplicaType(pods []*v1.Pod, rtypes []apiv1.ReplicaType) map[apiv1.ReplicaType][]*v1.Pod {
	var (
		sortedPods = make(map[apiv1.ReplicaType][]*v1.Pod)
		collectors = make(map[string]func(pod *v1.Pod))
	)
	for _, rtype := range rtypes {
		rt := rtype
		// replica-type label value is a lower cased replica type string.
		t := strings.ToLower(string(rt))
		collectors[t] = func(pod *v1.Pod) {
			// collect pod with matched replica type.
			sortedPods[rt] = append(sortedPods[rt], pod)
		}
	}
	for _, pod := range pods {
		rt := pod.Labels[apiv1.ReplicaTypeLabel]
		if collector := collectors[rt]; collector != nil {
			collector(pod)
		}
	}
	return sortedPods
}

var phaseCodes = map[v1.PodPhase]int{
	v1.PodPending:   0,
	v1.PodRunning:   1,
	v1.PodSucceeded: 2,
	v1.PodFailed:    2, // PodFailed has the same code as PodSucceeded because both of them are finished state.
	v1.PodUnknown:   -1,
}

// phaseComparator compares two pod phases and return comparison represented by an integer.
// return > 0:  phase p1 is ahead of p2.
// return == 0: phase p1 is at same stage as p2.
// return < 0:  phase p1 is behind of p2.
func phaseComparator(p1, p2 v1.PodPhase) int {
	return phaseCodes[p1] - phaseCodes[p2]
}
