package api

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/alibaba/kubedl/console/backend/pkg/utils"

	"github.com/gin-gonic/gin"

	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
)

func NewDataAPIsController() *dataAPIsController {
	return &dataAPIsController{
		dataHandler: handlers.NewDataHandler(),
	}
}

type dataAPIsController struct {
	dataHandler *handlers.DataHandler
}

func (dc *dataAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	overview := routes.Group("/data")
	overview.GET("/total", dc.getClusterTotal)
	overview.GET("/request/:podPhase", dc.getClusterRequest)
	overview.GET("/nodeInfos", dc.getClusterNodeInfos)
}

func (dc *dataAPIsController) getClusterTotal(c *gin.Context) {
	clusterTotal, err := dc.dataHandler.GetClusterTotalResource()
	if err != nil {
		Error(c, fmt.Sprintf("failed to getClusterTotal, err=%v", err))
		return
	}
	utils.Succeed(c, clusterTotal)
}

func (dc *dataAPIsController) getClusterRequest(c *gin.Context) {
	podPhase := c.Param("podPhase")
	if podPhase == "" {
		podPhase = string(corev1.PodRunning)
	}
	clusterRequest, err := dc.dataHandler.GetClusterRequestResource(podPhase)
	if err != nil {
		Error(c, fmt.Sprintf("failed to getClusterRequest, err=%v", err))
		return
	}
	utils.Succeed(c, clusterRequest)
}

func (dc *dataAPIsController) getClusterNodeInfos(c *gin.Context) {
	clusterNodeInfos, err := dc.dataHandler.GetClusterNodeInfos()
	if err != nil {
		Error(c, fmt.Sprintf("failed to getClusterNodeInfos, err=%v", err))
		return
	}
	utils.Succeed(c, clusterNodeInfos)
}
