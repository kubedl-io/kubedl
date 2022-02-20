/*
Copyright 2019 The Alibaba Authors.

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

package gang_schedule

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// GangScheduler describe a abstract gang scheduler to implement job gang scheduling,
// this interface discards the details of different scheduler implementations and
// call the intermediate object GangEntity, represented by runtime.Object.
type GangScheduler interface {
	// CreateGang creates a new gang entity to submit one scheduling process, with specified
	// minNumber and select batch of pods by single or multiple label selector. The
	// implementation should handle the relative relationship between job, pod and gang
	// entity.
	CreateGang(job metav1.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, schedPolicy *apiv1.SchedulingPolicy) (runtime.Object, error)

	// BindPodToGang binds an object with named gang entity object.
	BindPodToGang(job metav1.Object, podSpec *corev1.PodTemplateSpec, gangEntity runtime.Object, rtype string) error

	// GetGang get gang entity instance from cluster by name and namespace.
	GetGang(name types.NamespacedName) (client.ObjectList, error)

	// DeleteGang deletes gang entity object from cluster and finish corresponding gang
	// scheduling process.
	DeleteGang(name types.NamespacedName) error

	// PluginName of gang scheduler implementation, user selectively enable gang-scheduling implementation
	// by specifying --gang-scheduler-name={PluginName} in startup flags.
	PluginName() string

	// SchedulerName is the name of scheduler to dispatch pod, pod.spec.schedulerName will be
	// overridden when pod binds to gang-scheduler in BindPodToGang.
	SchedulerName() string
}

// NewGangScheduler receive a client as init parameter and return a new gang scheduler.
type NewGangScheduler func(mgr controllerruntime.Manager) GangScheduler
