package api

import (
	"flag"
	"fmt"

	"github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	v1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"k8s.io/klog"

	"github.com/gin-gonic/gin"

	"github.com/alibaba/kubedl/console/backend/pkg/utils"
)

func init() {
	flag.BoolVar(&detectJobInNS, "detect-job-in-ns", true, "detect jobs in namespace when do listing and return a map")
}

var (
	detectJobInNS bool
)

func NewKubeDLAPIsController() *KubeDLAPIsController {
	return &KubeDLAPIsController{
		handler: handlers.NewKubeDLHandler(),
	}
}

func (dc *KubeDLAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	kubedlAPIs := routes.Group("/kubedl")
	kubedlAPIs.GET("/images", dc.getImages)
	kubedlAPIs.GET("/namespaces", dc.getAvailableNamespaces)
}

type KubeDLAPIsController struct {
	handler *handlers.KubeDLHandler
}

func (dc *KubeDLAPIsController) getImages(c *gin.Context) {
	imageConfig := dc.handler.GetImageConfig()
	utils.Succeed(c, imageConfig)
}

func (dc *KubeDLAPIsController) getAvailableNamespaces(c *gin.Context) {
	avaliableNS, nsList, err := dc.handler.ListAvailableNamespaces()
	if err != nil {
		Error(c, fmt.Sprintf("failed to list avaliable namespaces, err: %v", err))
		return
	}
	if !detectJobInNS {
		utils.Succeed(c, avaliableNS)
		return
	}

	var nsWithWorkloads []string
	nsMap := make(map[string]bool)

	// find ns with workloads
	for _, ns := range avaliableNS {
		if dc.handler.DetectJobsInNS(ns, v1.TFJobKind) ||
			dc.handler.DetectJobsInNS(ns, v1.PyTorchJobKind) ||
			dc.handler.DetectJobsInNS(ns, v1alpha1.NotebookKind) {
			nsMap[ns] = true
			nsWithWorkloads = append(nsWithWorkloads, ns)
		}
	}

	// find ns with kubedl.io/workspace-name set
	for _, ns := range nsList.Items {
		if _, ok := nsMap[ns.Name]; !ok {
			if utils.IsKubedlManagedNamespace(&ns) {
				nsWithWorkloads = append(nsWithWorkloads, ns.Name)
			}
		}
	}
	klog.Infof("workspace list: %s ", nsWithWorkloads)
	utils.Succeed(c, nsWithWorkloads)
}
