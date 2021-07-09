package api

import (
	"flag"
	"fmt"
	v1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/console/backend/pkg/handlers"

	"github.com/alibaba/kubedl/console/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

func init() {
	flag.BoolVar(&detectJobInNS, "detect-job-in-ns", false, "detect jobs in namespace when do listing and return a map")
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
	kubedlAPIs.GET("/common-config", dc.getCommonConfig)
	kubedlAPIs.GET("/namespaces", dc.getAvailableNamespaces)
}

type KubeDLAPIsController struct {
	handler *handlers.KubeDLHandler
}

func (dc *KubeDLAPIsController) getCommonConfig(c *gin.Context) {
	kubedlCfg, err := dc.handler.GetKubeDLConfig()
	if err != nil {
		Error(c, fmt.Sprintf("failed to marshal kubedl common config, err: %v", err))
		return
	}

	utils.Succeed(c, kubedlCfg)
}

func (dc *KubeDLAPIsController) getAvailableNamespaces(c *gin.Context) {
	avaliableNS, err := dc.handler.ListAvailableNamespaces()
	if err != nil {
		Error(c, fmt.Sprintf("failed to list avaliable namespaces, err: %v", err))
		return
	}
	if !detectJobInNS {
		utils.Succeed(c, avaliableNS)
		return
	}

	nsWithJob := make(map[string]bool)
	for _, ns := range avaliableNS {
		nsWithJob[ns] = dc.handler.DetectJobsInNS(ns, v1.TFJobKind) ||
			dc.handler.DetectJobsInNS(ns, v1.PyTorchJobKind)
	}
	utils.Succeed(c, nsWithJob)
}
