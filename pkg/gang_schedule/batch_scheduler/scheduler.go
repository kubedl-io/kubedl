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

package batch_scheduler

import (
	"context"

	"github.com/alibaba/kubedl/api"
	"github.com/alibaba/kubedl/pkg/gang_schedule"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	"github.com/kubernetes-sigs/kube-batch/pkg/apis/scheduling/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Add to runtime scheme so that reflector of go-client will identify this CRD
	// controlled by scheduler.
	api.AddToSchemes = append(api.AddToSchemes, v1alpha1.AddToScheme)
}

func NewKubeBatchScheduler(mgr controllerruntime.Manager) gang_schedule.GangScheduler {
	return &kubeBatchScheduler{client: mgr.GetClient()}
}

var _ gang_schedule.GangScheduler = &kubeBatchScheduler{}

type kubeBatchScheduler struct {
	client client.Client
}

func (kbs *kubeBatchScheduler) Name() string {
	return "kube-batch"
}

func (kbs *kubeBatchScheduler) CreateGang(job metav1.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) (runtime.Object, error) {
	// Initialize pod group.
	podGroup := &v1alpha1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.GetName(),
			Namespace: job.GetNamespace(),
		},
		Spec: v1alpha1.PodGroupSpec{
			MinMember: k8sutil.GetTotalReplicas(replicas),
		},
	}
	// Extract api version and kind information from job.
	accessor, err := meta.TypeAccessor(job)
	if err != nil {
		return nil, err
	}
	apiVersion := accessor.GetAPIVersion()
	kind := accessor.GetKind()

	// Inject binding relationship into pod group by append owner reference.
	gang_schedule.AppendOwnerReference(podGroup, metav1.OwnerReference{
		APIVersion:         apiVersion,
		Kind:               kind,
		Name:               job.GetName(),
		UID:                job.GetUID(),
		BlockOwnerDeletion: pointer.BoolPtr(true),
		Controller:         pointer.BoolPtr(true),
	})

	err = kbs.client.Create(context.Background(), podGroup)
	return podGroup, err
}

func (kbs *kubeBatchScheduler) BindPodToGang(obj metav1.Object, entity runtime.Object) error {
	podSpec := obj.(*v1.PodTemplateSpec)
	// The newly-created pods should be submitted to target gang scheduler.
	if podSpec.Spec.SchedulerName == "" || podSpec.Spec.SchedulerName != kbs.Name() {
		podSpec.Spec.SchedulerName = kbs.Name()
	}
	return nil
}

func (kbs *kubeBatchScheduler) GetGang(name types.NamespacedName) (runtime.Object, error) {
	podGroup := &v1alpha1.PodGroup{}
	if err := kbs.client.Get(context.Background(), name, podGroup); err != nil {
		return nil, err
	}
	return podGroup, nil
}

func (kbs *kubeBatchScheduler) DeleteGang(name types.NamespacedName) error {
	podGroup, err := kbs.GetGang(name)
	if err != nil {
		return err
	}
	err = kbs.client.Delete(context.Background(), podGroup)
	// Discard deleted pod group object.
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	return err
}
