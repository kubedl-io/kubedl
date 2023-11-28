package runtime

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

var _ v1.ElasticScaling = &EmptyScaleImpl{}

// EmptyScaleImpl implements ElasticScaling interface but actually does no-ops.
type EmptyScaleImpl struct{}

func (e EmptyScaleImpl) EnableElasticScaling(job client.Object, runPolicy *v1.RunPolicy) bool {
	return false
}

func (e EmptyScaleImpl) ScaleOut(job client.Object, replicas map[v1.ReplicaType]*v1.ReplicaSpec, activePods []*corev1.Pod, activeServices []*corev1.Service) error {
	return nil
}

func (e EmptyScaleImpl) ScaleIn(job client.Object, replicas map[v1.ReplicaType]*v1.ReplicaSpec, activePods []*corev1.Pod, activeServices []*corev1.Service) error {
	return nil
}

func (e EmptyScaleImpl) CheckpointIfNecessary(job client.Object, activePods []*corev1.Pod) (completed bool, err error) {
	return true, nil
}
