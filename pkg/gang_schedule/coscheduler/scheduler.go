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

package coscheduler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	quotav1 "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/scheduler-plugins/pkg/apis/scheduling/v1alpha1"

	"github.com/alibaba/kubedl/apis"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/gang_schedule"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	resourceutils "github.com/alibaba/kubedl/pkg/util/resource_utils"
)

func init() {
	// Add to runtime scheme so that reflector of go-client will identify this CRD
	// controlled by scheduler.
	apis.AddToSchemes = append(apis.AddToSchemes, v1alpha1.AddToScheme)
}

const (
	LabelPodGroupIdentity     = "pod-group.scheduling.sigs.k8s.io"
	LabelPodGroupName         = "pod-group.scheduling.sigs.k8s.io/name"
	LabelPodGroupMinAvailable = "pod-group.scheduling.sigs.k8s.io/min-available"
)

func NewKubeCoscheduler(mgr controllerruntime.Manager) gang_schedule.GangScheduler {
	return &kubeCoscheduler{client: mgr.GetClient()}
}

var _ gang_schedule.GangScheduler = &kubeCoscheduler{}

type kubeCoscheduler struct {
	client client.Client
}

func (kbs *kubeCoscheduler) PluginName() string {
	return "kube-coscheduler"
}

func (kbs *kubeCoscheduler) SchedulerName() string {
	return "default-scheduler"
}

func (kbs *kubeCoscheduler) CreateGang(job metav1.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, schedPolicy *apiv1.SchedulingPolicy) (runtime.Object, error) {
	accessor, err := meta.TypeAccessor(job)
	if err != nil {
		return nil, err
	}

	var (
		apiVersion = accessor.GetAPIVersion()
		kind       = accessor.GetKind()
		podGroups  *v1alpha1.PodGroupList
	)

	// If DAG scheduling is enabled, kubedl will create individual gang entity for each role
	// to represent a separate stage, otherwise gang entity will be created in job granularity.
	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		podGroups = kbs.generateGangByRoleUnit(apiVersion, kind, job.GetName(), job.GetNamespace(), job.GetUID(), replicas)
	} else {
		podGroups = kbs.generateGangByJobUnit(apiVersion, kind, job.GetName(), job.GetNamespace(), job.GetUID(), replicas, schedPolicy)
	}

	for i := range podGroups.Items {
		pg := &podGroups.Items[i]
		err = kbs.client.Get(context.Background(), types.NamespacedName{Name: pg.Name, Namespace: pg.Namespace}, &v1alpha1.PodGroup{})
		if err != nil && errors.IsNotFound(err) {
			err = kbs.client.Create(context.Background(), pg)
		}
		if err != nil {
			return nil, err
		}
	}

	return podGroups, err
}

func (kbs *kubeCoscheduler) BindPodToGang(job metav1.Object, podSpec *v1.PodTemplateSpec, gangEntity runtime.Object, rtype string) error {
	if rtype == strings.ToLower(string(apiv1.JobReplicaTypeAIMaster)) {
		podSpec.Spec.SchedulerName = "default-scheduler"
		return nil
	}
	podGroups, ok := gangEntity.(*v1alpha1.PodGroupList)
	if !ok {
		klog.Warningf("gang entity cannot convert to gang list, entity: %+v", gangEntity)
		return nil
	}
	if len(podGroups.Items) == 0 {
		return fmt.Errorf("unexpected empty gang entity list, job name: %s", job.GetName())
	}

	matchLabels := map[string]string{apiv1.LabelGangSchedulingJobName: job.GetName()}
	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		matchLabels[apiv1.ReplicaTypeLabel] = rtype
	}
	selector := labels.SelectorFromSet(matchLabels)

	for i := range podGroups.Items {
		pg := &podGroups.Items[i]
		if pg.Labels != nil && selector.Matches(labels.Set(pg.Labels)) {
			gang_schedule.AppendOwnerReference(podSpec, metav1.OwnerReference{
				APIVersion:         pg.APIVersion,
				Kind:               pg.Kind,
				Name:               pg.Name,
				UID:                pg.UID,
				Controller:         pointer.BoolPtr(false),
				BlockOwnerDeletion: pointer.BoolPtr(true),
			})

			if podSpec.Labels == nil {
				podSpec.Labels = map[string]string{}
			}
			// For PodGroup CRD based coscheduling protocol.
			podSpec.Labels[LabelPodGroupIdentity] = pg.Name
			// For label based lightweight coscheduling protocol.
			podSpec.Labels[LabelPodGroupName] = pg.Name
			podSpec.Labels[LabelPodGroupMinAvailable] = strconv.Itoa(int(pg.Spec.MinMember))
			break
		}
	}

	return nil
}

func (kbs *kubeCoscheduler) GetGang(name types.NamespacedName) (client.ObjectList, error) {
	podGroups := &v1alpha1.PodGroupList{}
	err := kbs.client.List(context.Background(), podGroups, client.MatchingLabels{
		apiv1.LabelGangSchedulingJobName: name.Name,
	}, client.InNamespace(name.Namespace))
	if err != nil {
		return nil, err
	}
	return podGroups, nil
}

func (kbs *kubeCoscheduler) DeleteGang(name types.NamespacedName) error {
	pgs, err := kbs.GetGang(name)
	if err != nil {
		return err
	}
	podGroups := pgs.(*v1alpha1.PodGroupList)

	for i := range podGroups.Items {
		pg := &podGroups.Items[i]
		if err = kbs.client.Delete(context.Background(), pg); err != nil {
			return err
		}
	}
	return err
}

func (kbs *kubeCoscheduler) generateGangByJobUnit(apiVersion, kind, name, namespace string, uid types.UID, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, schedPolicy *apiv1.SchedulingPolicy) *v1alpha1.PodGroupList {
	pg := v1alpha1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{apiv1.LabelGangSchedulingJobName: name},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         apiVersion,
					Kind:               kind,
					Name:               name,
					UID:                uid,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
			},
		},
		Spec: v1alpha1.PodGroupSpec{MinMember: k8sutil.GetTotalReplicas(replicas)},
	}
	jobResource, _ := resourceutils.JobResourceRequests(replicas)

	if aimaster := replicas[apiv1.JobReplicaTypeAIMaster]; aimaster != nil && aimaster.Replicas != nil {
		if *aimaster.Replicas > 0 {
			pg.Spec.MinMember -= *aimaster.Replicas
			jobResource = quotav1.SubtractWithNonNegativeResult(jobResource,
				resourceutils.Multiply(int64(*aimaster.Replicas), resourceutils.ReplicaResourceRequests(aimaster)))
		}
	}

	if schedPolicy != nil && schedPolicy.MinAvailable != nil && *schedPolicy.MinAvailable > 0 {
		pg.Spec.MinMember = *schedPolicy.MinAvailable
	}

	pg.Spec.MinResources = &jobResource
	return &v1alpha1.PodGroupList{Items: []v1alpha1.PodGroup{pg}}
}

func (kbs *kubeCoscheduler) generateGangByRoleUnit(apiVersion, kind, name, namespace string, uid types.UID, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) *v1alpha1.PodGroupList {
	pgs := v1alpha1.PodGroupList{Items: make([]v1alpha1.PodGroup, 0, len(replicas))}

	for rtype, spec := range replicas {
		if rtype == apiv1.JobReplicaTypeAIMaster {
			continue
		}
		rt := strings.ToLower(string(rtype))
		gangName := fmt.Sprintf("%s-%s", name, rt)
		resources := resourceutils.ReplicaResourceRequests(spec)
		pgs.Items = append(pgs.Items, v1alpha1.PodGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gangName,
				Namespace: namespace,
				Labels: map[string]string{
					apiv1.LabelGangSchedulingJobName: name,
					apiv1.ReplicaTypeLabel:           rt,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         apiVersion,
						Kind:               kind,
						Name:               name,
						UID:                uid,
						Controller:         pointer.BoolPtr(true),
						BlockOwnerDeletion: pointer.BoolPtr(true),
					},
				},
			},
			Spec: v1alpha1.PodGroupSpec{
				MinMember:    *spec.Replicas,
				MinResources: &resources,
			},
		})
	}
	return &pgs
}
