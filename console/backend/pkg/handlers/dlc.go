package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/alibaba/kubedl/console/backend/pkg/constants"
	utils "github.com/alibaba/kubedl/pkg/storage/backends/client"
	v1 "k8s.io/api/core/v1"
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
		return nil, errors.New("empty common configmap name")
	}

	cm := v1.ConfigMap{}
	if err := h.client.Get(context.Background(), types.NamespacedName{
		Name:      constants.ConfigMapName,
		Namespace: constants.DLCSystemNamespace,
	}, &cm); err != nil {
		return nil, fmt.Errorf("failed to get common config, err: %v", err)
	}

	commonCfg := cm.Data["commonConfig"]
	dlcCommonCfg := DLCCommonConfig{}
	if err := json.Unmarshal([]byte(commonCfg), &dlcCommonCfg); err != nil {
		return nil, fmt.Errorf("failed to marshal dlc common config, err: %v", err)
	}

	return &dlcCommonCfg, nil
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
