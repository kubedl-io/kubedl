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


func NewDLCHandler() *DLCHandler {
	return &DLCHandler{client: utils.GetCtrlClient()}
}

type DLCHandler struct {
	client client.Client
}

type DLCCommonConfig struct {
	Namespace        string   `json:"namespace"`
	TFCpuImages      []string `json:"pai-tf-cpu-images"`
	TFGpuImages      []string `json:"pai-tf-gpu-images"`
	PytorchGpuImages []string `json:"pai-pytorch-gpu-images"`
	ClusterID        string   `json:"clusterId,omitempty"`
	DLCVersion       string   `json:"dlcVersion,omitempty"`
}

func (h *DLCHandler) GetDLCConfig() (*DLCCommonConfig, error) {
	if constants.ConfigMapName == "" {
		return nil, fmt.Errorf("empty common configmap name")
	}

	cm := &v1.ConfigMap{}
	err := h.client.Get(context.Background(), types.NamespacedName{
		Namespace: constants.DLCSystemNamespace,
		Name:      constants.ConfigMapName,
	}, cm)
	// Create initial ConfigMap if not exists
	if errors.IsNotFound(err) {
		cm, err = h.createDLCConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create common config, err: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get common config, err: %v", err)
	}

	commonCfg := cm.Data["commonConfig"]
	dlcCommonCfg := DLCCommonConfig{}
	if err := json.Unmarshal([]byte(commonCfg), &dlcCommonCfg); err != nil {
		return nil, fmt.Errorf("failed to marshal dlc common config, err: %v", err)
	}

	return &dlcCommonCfg, nil
}

func (h *DLCHandler) createDLCConfig() (*v1.ConfigMap, error) {
	commonConfig := DLCCommonConfig{
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
			Namespace: constants.DLCSystemNamespace,
			Name:      DatasourceConfigMapName,
		},
		Data: data,
	}
	if err := h.client.Create(context.TODO(), initConfigMap); err != nil {
		klog.Errorf("Failed to create ConfigMap, ns: %s, name: %s, err: %v", constants.DLCSystemNamespace, constants.ConfigMapName, err)
		return nil, err
	}
	return initConfigMap, nil
}

func (h *DLCHandler) ListAvailableNamespaces() ([]string, error) {
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

func (h *DLCHandler) DetectJobsInNS(ns, kind string) bool {
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
