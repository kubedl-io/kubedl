package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	backendutils "github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/tensorboard"
	networkingv1 "k8s.io/api/networking/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTensorBoardHandler() *TensorBoardHandler {
	return &TensorBoardHandler{client: backendutils.GetCtrlClient()}
}

type TensorBoardHandler struct {
	client client.Client
}

func (th *TensorBoardHandler) GetTensorBoardInstance(jobNamespace, jobName, jobUID string) (*v1.Pod, error) {
	pods := v1.PodList{}
	err := th.client.List(context.Background(), &pods, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			apiv1.ReplicaTypeLabel: "tensorboard",
			apiv1.JobNameLabel:     jobName,
		}),
		Namespace: jobNamespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tensorboard pods, err: %v", err)
	}

	var tb *v1.Pod
	for idx := range pods.Items {
		pod := &pods.Items[idx]
		if validateOwnerReference(pod.OwnerReferences, jobName, jobUID) {
			tb = pod
			break
		}
	}

	if tb != nil {
		klog.Infof("get tensorboard instance[%s] of job %s/%s", tb.Name, jobNamespace, jobName)
	} else {
		klog.Infof("no tensorboard instance found of job %s/%s", jobNamespace, jobName)
	}
	return tb, nil
}

func (th *TensorBoardHandler) ListIngressInstances(jobNamespace, jobName, jobUID string) ([]*networkingv1.Ingress, error) {
	ingresses := networkingv1.IngressList{}
	err := th.client.List(context.Background(), &ingresses, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			apiv1.ReplicaTypeLabel: "tensorboard",
			apiv1.JobNameLabel:     jobName,
		}),
		Namespace: jobNamespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses, err: %v", err)
	}

	// There may multiple ingress belongs to one tensorboard instance.
	validIngresses := make([]*networkingv1.Ingress, 0)
	for idx := range ingresses.Items {
		ingress := &ingresses.Items[idx]
		if validateOwnerReference(ingress.OwnerReferences, jobName, jobUID) {
			validIngresses = append(validIngresses, ingress)
		}
	}

	return validIngresses, nil
}

func (th *TensorBoardHandler) GetJob(jobNamespace, jobName, kind string) (metav1.Object, error) {
	job := utils.InitJobRuntimeObjectByKind(kind)
	err := th.client.Get(context.Background(), types.NamespacedName{
		Namespace: jobNamespace,
		Name:      jobName,
	}, job)
	if err != nil {
		return nil, fmt.Errorf("failed to get job object, err: %v", err)
	}

	metaObj, ok := utils.RuntimeObjToMetaObj(job)
	if !ok {
		return nil, fmt.Errorf("job [%s/%s] is not a meta object", jobNamespace, jobName)
	}
	return metaObj, nil
}

func (th *TensorBoardHandler) ApplyNewTensorBoardConfig(jobNamespace, jobName, kind string, config tensorboard.TensorBoard) error {
	// Set default TTL seconds.
	if config.TTLSecondsAfterJobFinished == 0 {
		config.TTLSecondsAfterJobFinished = 60 * 60
	}

	job := utils.InitJobRuntimeObjectByKind(kind)
	err := th.client.Get(context.Background(), types.NamespacedName{
		Namespace: jobNamespace,
		Name:      jobName,
	}, job)
	if err != nil {
		return fmt.Errorf("failed to get job object, err: %v", err)
	}

	cfgBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	anno := job.(metav1.Object).GetAnnotations()
	if anno == nil {
		anno = make(map[string]string)
	}
	anno[apiv1.AnnotationTensorBoardConfig] = string(cfgBytes)
	job.(metav1.Object).SetAnnotations(anno)
	/*
		patch := fmt.Sprintf(
			`{"metadata":{"annotations":{"%s":"%s"}}}`, apiv1.AnnotationTensorBoardConfig, string(cfgBytes),
		)
	*/

	return th.client.Update(context.Background(), job)
}

func validateOwnerReference(ownerRefs []metav1.OwnerReference, ownerName, ownerUID string) bool {
	for _, ownerRef := range ownerRefs {
		if ownerRef.Name == ownerName && string(ownerRef.UID) == ownerUID {
			return true
		}
	}
	return false
}
