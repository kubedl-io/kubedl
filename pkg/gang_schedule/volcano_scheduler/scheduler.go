package volcano_scheduler

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/utils/pointer"

	"github.com/alibaba/kubedl/apis"
	"github.com/alibaba/kubedl/pkg/gang_schedule"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
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

func (vs *volcanoScheduler) Name() string {
	return "volcano"
}

func (vs *volcanoScheduler) CreateGang(job metav1.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) (runtime.Object, error) {
	// Initialize pod group.
	podGroup := &v1beta1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.GetName(),
			Namespace: job.GetNamespace(),
		},
		Spec: v1beta1.PodGroupSpec{
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

	err = vs.client.Create(context.Background(), podGroup)
	return podGroup, err
}

func (vs *volcanoScheduler) BindPodToGang(obj metav1.Object, entity runtime.Object) error {
	podSpec := obj.(*v1.PodTemplateSpec)
	podGroup := entity.(*v1beta1.PodGroup)
	// The newly-created pods should be submitted to target gang scheduler.
	if podSpec.Spec.SchedulerName == "" || podSpec.Spec.SchedulerName != vs.Name() {
		podSpec.Spec.SchedulerName = vs.Name()
	}
	if podSpec.Annotations == nil {
		podSpec.Annotations = map[string]string{}
	}
	podSpec.Annotations[v1beta1.KubeGroupNameAnnotationKey] = podGroup.GetName()
	return nil
}

func (vs *volcanoScheduler) GetGang(name types.NamespacedName) (runtime.Object, error) {
	podGroup := &v1beta1.PodGroup{}
	if err := vs.client.Get(context.Background(), name, podGroup); err != nil {
		return nil, err
	}
	return podGroup, nil
}

func (vs *volcanoScheduler) DeleteGang(name types.NamespacedName) error {
	podGroup, err := vs.GetGang(name)
	if err != nil {
		return err
	}
	err = vs.client.Delete(context.Background(), podGroup)
	// Discard deleted pod group object.
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	return err
}
