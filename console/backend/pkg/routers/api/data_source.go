package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
)

func NewDataSourceAPIsController(dataSourceHandler *handlers.DataSourceHandler) *dataSourceAPIsController {
	return &dataSourceAPIsController{
		dataSourceHandler: dataSourceHandler,
	}
}

type dataSourceAPIsController struct {
	dataSourceHandler *handlers.DataSourceHandler
}

func (dc *dataSourceAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	overview := routes.Group("/datasource")
	overview.POST("", dc.postDataSource)
	overview.DELETE("/:name", dc.deleteDataSource)
	overview.PUT("", dc.putDataSource)
	overview.GET("", dc.getDataSource)
	overview.GET("/:name", dc.getDataSource)
}

func (dc *dataSourceAPIsController) postDataSource(c *gin.Context) {
	userid := c.PostForm("userid")
	username := c.PostForm("username")
	namespace := c.PostForm("namespace")
	name := c.PostForm("name")
	_type := c.PostForm("type")
	pvcName := c.PostForm("pvc_name")
	localPath := c.PostForm("local_path")
	description := c.PostForm("description")
	createTime := time.Now().Format("2006-01-02 15:04:05")
	updateTime := time.Now().Format("2006-01-02 15:04:05")

	datasource := model.DataSource{
		UserId:      userid,
		Username:    username,
		Namespace:   namespace,
		Name:        name,
		Type:        _type,
		PvcName:     pvcName,
		LocalPath:   localPath,
		Description: description,
		CreateTime:  createTime,
		UpdateTime:  updateTime,
	}

	err := dc.dataSourceHandler.PostDataSourceToConfigMap(datasource)
	if err != nil {
		Error(c, fmt.Sprintf("failed to createDataSource, err=%v", err))
		return
	}
	utils.Succeed(c, "success to createDataSource")
}

func (dc *dataSourceAPIsController) deleteDataSource(c *gin.Context) {
	name := c.Param("name")
	err := dc.dataSourceHandler.DeleteDataSourceFromConfigMap(name)
	if err != nil {
		Error(c, fmt.Sprintf("failed to delete DataSource, err=%v", err))
		return
	}
	utils.Succeed(c, "success to delete DataSource.")
}

func (dc *dataSourceAPIsController) putDataSource(c *gin.Context) {
	userId := c.PostForm("userid")
	username := c.PostForm("username")
	namespace := c.PostForm("namespace")
	name := c.PostForm("name")
	_type := c.PostForm("type")
	pvcName := c.PostForm("pvc_name")
	localPath := c.PostForm("local_path")
	description := c.PostForm("description")
	updateTime := time.Now().Format("2006-01-02 15:04:05")

	datasource := model.DataSource{
		UserId:      userId,
		Username:    username,
		Namespace:   namespace,
		Name:        name,
		Type:        _type,
		PvcName:     pvcName,
		LocalPath:   localPath,
		Description: description,
		UpdateTime:  updateTime,
	}

	err := dc.dataSourceHandler.PutDataSourceToConfigMap(datasource)
	if err != nil {
		Error(c, fmt.Sprintf("failed to putDataSource, err=%v", err))
		return
	}
	utils.Succeed(c, "success to putDataSource")
}

func (dc *dataSourceAPIsController) getDataSource(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		dataSources, err := dc.dataSourceHandler.ListDataSourceFromConfigMap()
		if err != nil {
			Error(c, fmt.Sprintf("failed to getDataSource, err=%v", err))
			return
		}
		utils.Succeed(c, dataSources)
	} else {
		dataSources, err := dc.dataSourceHandler.GetDataSourceFromConfigMap(name)
		if err != nil {
			Error(c, fmt.Sprintf("failed to getDataSource, err=%v", err))
			return
		}
		utils.Succeed(c, dataSources)
	}

}
