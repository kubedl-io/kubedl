package handlers

import (
	"context"
	"encoding/json"
	"k8s.io/klog"

	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/console/backend/pkg/constants"
	utils "github.com/alibaba/kubedl/pkg/storage/backends/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)


func NewKubeDLHandler() *KubeDLHandler {
	return &KubeDLHandler{client: utils.GetCtrlClient()}
}

type KubeDLHandler struct {
	client client.Client
}

type KubeDLCommonConfig struct {
	Namespace        string   `json:"namespace"`
	TFCpuImages      []string `json:"tf-cpu-images"`
	TFGpuImages      []string `json:"tf-gpu-images"`
	PytorchGpuImages []string `json:"pytorch-gpu-images"`
	KubeDLVersion    string   `json:"kubedlVersion,omitempty"`
}

func (h *KubeDLHandler) GetKubeDLConfig() (*KubeDLCommonConfig, error) {
	if constants.ConfigMapName == "" {
		return nil, fmt.Errorf("empty common configmap name")
	}

	cm := &v1.ConfigMap{}
	err := h.client.Get(context.Background(), types.NamespacedName{
		Namespace: constants.KubeDLSystemNamespace,
		Name:      constants.ConfigMapName,
	}, cm)
	// Create initial ConfigMap if not exists
	if errors.IsNotFound(err) {
		cm, err = h.createKubeDLConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create common config, err: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get common config, err: %v", err)
	}

	commonCfg := cm.Data["commonConfig"]
	kubeDLCommonCfg := KubeDLCommonConfig{}
	if err := json.Unmarshal([]byte(commonCfg), &kubeDLCommonCfg); err != nil {
		return nil, fmt.Errorf("failed to marshal kubedl common config, err: %v", err)
	}

	return &kubeDLCommonCfg, nil
}

func (h *KubeDLHandler) createKubeDLConfig() (*v1.ConfigMap, error) {
	commonConfig := KubeDLCommonConfig{
		Namespace:   "kubedl",
		TFCpuImages: []string{
			"kubedl/tf-mnist-with-summaries:1.0",
		},
		PytorchGpuImages: []string{
			"kubedl/pytorch-dist-example",
		},
	}
	commonConfigBytes, err := json.Marshal(commonConfig)
	if err != nil {
		return nil, err
	}
	data := make(map[string]string)
	data["commonConfig"] = string(commonConfigBytes)

	initConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.KubeDLSystemNamespace,
			Name:      constants.ConfigMapName,
		},
		Data: data,
	}
	if err := h.client.Create(context.TODO(), initConfigMap); err != nil {
		klog.Errorf("Failed to create ConfigMap, ns: %s, name: %s, err: %v", constants.KubeDLSystemNamespace, constants.ConfigMapName, err)
		return nil, err
	}
	return initConfigMap, nil
}

func (h *KubeDLHandler) ListAvailableNamespaces() ([]string, error) {
	namespaces := v1.NamespaceList{}
	if err := h.client.List(context.Background(), &namespaces); err != nil {
		return nil, err
	}

	preservedNS := map[string]bool{
		"kube-system": true,
	}

	var available []string
	for _, item := range namespaces.Items {
		if _, ok := preservedNS[item.Name]; ok {
			continue
		}
		available = append(available, item.Name)
	}
	return available, nil
}

func (h *KubeDLHandler) DetectJobsInNS(ns, kind string) bool {
	var (
		list     runtime.Object
		detector func(object runtime.Object) bool
	)

	switch kind {
	case v1alpha1.TFJobKind:
		list = &v1alpha1.TFJobList{}
		detector = func(object runtime.Object) bool {
			tfJobs := object.(*v1alpha1.TFJobList)
			return len(tfJobs.Items) > 0
		}
	case v1alpha1.PyTorchJobKind:
		list = &v1alpha1.PyTorchJobList{}
		detector = func(object runtime.Object) bool {
			pytorchJobs := object.(*v1alpha1.PyTorchJobList)
			return len(pytorchJobs.Items) > 0
		}
	default:
		return false
	}

	if err := h.client.List(context.Background(), list, &client.ListOptions{Namespace: ns}); err != nil {
		return false
	}
	return detector(list)
}
