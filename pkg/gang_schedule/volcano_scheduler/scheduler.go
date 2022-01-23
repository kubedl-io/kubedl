package volcano_scheduler

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	"k8s.io/utils/pointer"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/alibaba/kubedl/apis"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/gang_schedule"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
)

func init() {
	// Add to runtime scheme so that reflector of go-client will identify this CRD
	// controlled by scheduler.
	apis.AddToSchemes = append(apis.AddToSchemes, v1beta1.AddToScheme)
}

func NewVolcanoScheduler(mgr controllerruntime.Manager) gang_schedule.GangScheduler {
	return &volcanoScheduler{client: mgr.GetClient()}
}

var _ gang_schedule.GangScheduler = &volcanoScheduler{}

type volcanoScheduler struct {
	client client.Client
}

func (vs *volcanoScheduler) PluginName() string {
	return "volcano"
}

func (vs *volcanoScheduler) SchedulerName() string {
	return "volcano"
}

func (vs *volcanoScheduler) CreateGang(job metav1.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, schedPolicy *apiv1.SchedulingPolicy) (runtime.Object, error) {
	// Extract api version and kind information from job.
	accessor, err := meta.TypeAccessor(job)
	if err != nil {
		return nil, err
	}

	var (
		apiVersion = accessor.GetAPIVersion()
		kind       = accessor.GetKind()
		podGroups  *v1beta1.PodGroupList
	)

	// If DAG scheduling is enabled, kubedl will create individual podgrpoups entity for each role
	// to represent a separate stage, otherwise podgrpoups entity will be created in job granularity.
	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		podGroups = vs.generateGangByRoleUnit(apiVersion, kind, job.GetName(), job.GetNamespace(), job.GetUID(), replicas)
	} else {
		podGroups = vs.generateGangByJobUnit(apiVersion, kind, job.GetName(), job.GetNamespace(), job.GetUID(), replicas, schedPolicy)
	}

	for i := range podGroups.Items {
		pg := &podGroups.Items[i]
		err = vs.client.Get(context.Background(), types.NamespacedName{Name: pg.Name, Namespace: pg.Namespace}, &v1beta1.PodGroup{})
		if err != nil && errors.IsNotFound(err) {
			err = vs.client.Create(context.Background(), pg)
		}
		if err != nil {
			return nil, err
		}
	}

	return podGroups, err
}

func (vs *volcanoScheduler) BindPodToGang(job metav1.Object, podSpec *v1.PodTemplateSpec, gangEntity runtime.Object, rtype string) error {
	podGroups, ok := gangEntity.(*v1beta1.PodGroupList)
	if !ok {
		klog.Warningf("podgrpoups entity cannot convert to podgrpoups list, entity: %+v", gangEntity)
		return nil
	}
	if len(podGroups.Items) == 0 {
		return fmt.Errorf("unexpected empty podgrpoups entity list, job name: %s", job.GetName())
	}

	podGroupName := job.GetName()
	matchLabels := map[string]string{apiv1.LabelGangSchedulingJobName: job.GetName()}
	if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) {
		matchLabels[apiv1.ReplicaTypeLabel] = rtype
		podGroupName = fmt.Sprintf("%s-%s", job.GetName(), rtype)
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
			podGroupName = pg.Name
			break
		}
	}

	if podSpec.Annotations == nil {
		podSpec.Annotations = map[string]string{}
	}
	podSpec.Annotations[v1beta1.KubeGroupNameAnnotationKey] = podGroupName
	return nil
}

func (vs *volcanoScheduler) GetGang(name types.NamespacedName) (runtime.Object, error) {
	podGroups := &v1beta1.PodGroupList{}
	if err := vs.client.List(context.Background(), podGroups, client.MatchingLabels{
		apiv1.LabelGangSchedulingJobName: name.Name,
	}, client.InNamespace(name.Namespace)); err != nil {
		return nil, err
	}
	return podGroups, nil
}

func (vs *volcanoScheduler) DeleteGang(name types.NamespacedName) error {
	pgs, err := vs.GetGang(name)
	if err != nil {
		return err
	}
	podGroups := pgs.(*v1beta1.PodGroupList)

	for i := range podGroups.Items {
		pg := &podGroups.Items[i]
		if err = vs.client.Delete(context.Background(), pg); err != nil {
			return err
		}
	}
	return err
}

func (vs *volcanoScheduler) generateGangByJobUnit(apiVersion, kind, name, namespace string, uid types.UID, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, schedPolicy *apiv1.SchedulingPolicy) *v1beta1.PodGroupList {
	pg := v1beta1.PodGroup{
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
		Spec: v1beta1.PodGroupSpec{MinMember: k8sutil.GetTotalReplicas(replicas)},
	}

	if schedPolicy != nil && schedPolicy.MinAvailable != nil && *schedPolicy.MinAvailable > 0 {
		pg.Spec.MinMember = *schedPolicy.MinAvailable
	}

	return &v1beta1.PodGroupList{Items: []v1beta1.PodGroup{pg}}
}

func (vs *volcanoScheduler) generateGangByRoleUnit(apiVersion, kind, name, namespace string, uid types.UID, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) *v1beta1.PodGroupList {
	pgs := v1beta1.PodGroupList{Items: make([]v1beta1.PodGroup, 0, len(replicas))}

	for rtype, spec := range replicas {
		rt := strings.ToLower(string(rtype))
		gangName := fmt.Sprintf("%s-%s", name, rt)
		pgs.Items = append(pgs.Items, v1beta1.PodGroup{
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
			Spec: v1beta1.PodGroupSpec{MinMember: *spec.Replicas},
		})
	}

	return &pgs
}
