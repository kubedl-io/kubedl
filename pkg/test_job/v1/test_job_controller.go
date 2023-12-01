package v1

import (
	"context"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/runtime"
)

var _ apiv1.ControllerInterface = &TestJobController{}

type TestJobController struct {
	Job      *TestJob
	Pods     []*corev1.Pod
	Services []*corev1.Service
	runtime.EmptyScaleImpl
}

func (in *TestJobController) GetNodeForModelOutput(pods []*corev1.Pod) (nodeName string) {
	return ""
}

func (t TestJobController) GetReconcileOrders() []apiv1.ReplicaType {
	return []apiv1.ReplicaType{
		TestReplicaTypeMaster,
		TestReplicaTypeWorker,
	}
}

func (t TestJobController) GetPodsForJob(job client.Object) ([]*corev1.Pod, error) {
	return []*corev1.Pod{}, nil
}

func (t TestJobController) GetServicesForJob(job client.Object) ([]*corev1.Service, error) {
	return []*corev1.Service{}, nil
}

func (TestJobController) ControllerName() string {
	return "test-operator"
}

func (TestJobController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersionKind
}

func (TestJobController) GetAPIGroupVersion() schema.GroupVersion {
	return SchemeGroupVersion
}

func (TestJobController) GetGroupNameLabelValue() string {
	return GroupName
}

func (TestJobController) GetJobRoleKey() string {
	return apiv1.JobRoleLabel
}

func (TestJobController) GetDefaultContainerPortName() string {
	return "default-port-name"
}

func (TestJobController) GetDefaultContainerPortNumber() int32 {
	return int32(9999)
}

func (t *TestJobController) GetJobFromInformerCache(namespace, name string) (client.Object, error) {
	return t.Job, nil
}

func (t *TestJobController) GetJobFromAPIClient(namespace, name string) (client.Object, error) {
	return t.Job, nil
}

func (t *TestJobController) DeleteJob(job interface{}) error {
	log.Info("Delete job")
	t.Job = nil
	return nil
}

func (t *TestJobController) UpdateJobStatus(job client.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, jobStatus *apiv1.JobStatus, restart bool) error {
	return nil
}

func (t *TestJobController) UpdateJobStatusInApiServer(job client.Object, jobStatus *apiv1.JobStatus) error {
	return nil
}

func (t *TestJobController) CreateService(job interface{}, service *corev1.Service) error {
	return nil
}

func (t *TestJobController) DeleteService(job interface{}, name string, namespace string) error {
	log.Info("Deleting service " + name)
	var remainingServices []*corev1.Service
	for _, tservice := range t.Services {
		if tservice.Name != name {
			remainingServices = append(remainingServices, tservice)
		}
	}
	t.Services = remainingServices
	return nil
}

func (t *TestJobController) CreatePod(job interface{}, pod *corev1.Pod) error {
	return nil
}

func (t *TestJobController) DeletePod(job interface{}, pod *corev1.Pod) error {
	log.Info("Deleting pod " + pod.Name)
	var remainingPods []*corev1.Pod
	for _, tpod := range t.Pods {
		if tpod.Name != pod.Name {
			remainingPods = append(remainingPods, tpod)
		}
	}
	t.Pods = remainingPods
	return nil
}

func (t *TestJobController) SetClusterSpec(ctx context.Context, job client.Object, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return nil
}

func (t *TestJobController) GetDefaultContainerName() string {
	return "default-container"
}

func (t *TestJobController) IsMasterRole(replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, rtype apiv1.ReplicaType, index int) bool {
	return true
}
