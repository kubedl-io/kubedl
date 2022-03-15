package handlers

import (
	"context"
	"encoding/json"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nbv1alpha1 "github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	utils "github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/constants"
)

func NewKubeDLHandler() *KubeDLHandler {
	return &KubeDLHandler{client: utils.GetCtrlClient()}
}

type KubeDLHandler struct {
	client client.Client
}

// ImageConfig contains the image for jobs and notebooks
type ImageConfig struct {
	TFCpuImages      []string `json:"tf-cpu-images"`
	TFGpuImages      []string `json:"tf-gpu-images"`
	PytorchGpuImages []string `json:"pytorch-gpu-images"`
	NotebookImages   []string `json:"notebook-images"`
}

func (h *KubeDLHandler) GetImageConfig() *ImageConfig {
	cm := &v1.ConfigMap{}
	err := h.client.Get(context.Background(), types.NamespacedName{
		Namespace: constants.KubeDLSystemNamespace,
		Name:      constants.KubeDLConsoleConfig,
	}, cm)
	if err != nil {
		glog.Infof("get image config error: %v", err)
		return &ImageConfig{}
	}

	commonCfg := cm.Data["images"]
	imageConfig := ImageConfig{}
	if err := json.Unmarshal([]byte(commonCfg), &imageConfig); err != nil {
		return &ImageConfig{}
	}
	return &imageConfig
}

func (h *KubeDLHandler) ListAvailableNamespaces() ([]string, *v1.NamespaceList, error) {
	namespaces := v1.NamespaceList{}
	if err := h.client.List(context.Background(), &namespaces); err != nil {
		return nil, nil, err
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
	return available, &namespaces, nil
}
func (h *KubeDLHandler) DetectJobsInNS(ns, kind string) bool {
	var (
		list     client.ObjectList
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
	case nbv1alpha1.NotebookKind:
		list = &nbv1alpha1.NotebookList{}
		detector = func(object runtime.Object) bool {
			notebooks := object.(*nbv1alpha1.NotebookList)
			return len(notebooks.Items) > 0
		}
	default:
		return false
	}

	if err := h.client.List(context.Background(), list, &client.ListOptions{Namespace: ns}); err != nil {
		return false
	}
	return detector(list)
}
