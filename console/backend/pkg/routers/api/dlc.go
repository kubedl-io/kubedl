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

func NewDLCAPIsController() *DLCAPIsController {
	return &DLCAPIsController{
		handler: handlers.NewDLCHandler(),
	}
}

func (dc *DLCAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	dlcAPIs := routes.Group("/dlc")
	dlcAPIs.GET("/common-config", dc.getCommonConfig)
	dlcAPIs.GET("/namespaces", dc.getAvailableNamespaces)
}

type DLCAPIsController struct {
	handler *handlers.DLCHandler
}

func (dc *DLCAPIsController) getCommonConfig(c *gin.Context) {
	dlcCfg, err := dc.handler.GetDLCConfig()
	if err != nil {
		Error(c, fmt.Sprintf("failed to marshal dlc common config, err: %v", err))
		return
	}

	utils.Succeed(c, dlcCfg)
}

func (dc *DLCAPIsController) getAvailableNamespaces(c *gin.Context) {
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
