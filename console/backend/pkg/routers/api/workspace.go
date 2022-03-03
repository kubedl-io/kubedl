package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
)

func NewWorkspaceAPIsController(objBackend backends.ObjectStorageBackend, dataSourceHandler *handlers.DataSourceHandler) *WorkspaceAPIsController {
	return &WorkspaceAPIsController{
		objBackend:        objBackend,
		dataSourceHandler: dataSourceHandler,
	}
}

// WorkspaceAPIsController for handling workspaces related HTTP verbs
type WorkspaceAPIsController struct {
	objBackend        backends.ObjectStorageBackend
	dataSourceHandler *handlers.DataSourceHandler
}

func (wc *WorkspaceAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	workspaceAPI := routes.Group("/workspace")
	workspaceAPI.POST("/create", wc.CreateWorkspace)
	workspaceAPI.GET("/list", wc.ListWorkspaces)
	workspaceAPI.DELETE("/:name", wc.DeleteWorkspace)
	workspaceAPI.GET("/detail", wc.GetWorkspaceDetail)
}

func (wc *WorkspaceAPIsController) CreateWorkspace(c *gin.Context) {

	username := c.PostForm("username")
	namespace := c.PostForm("namespace")
	name := c.PostForm("name")
	_type := c.PostForm("type")
	pvcName := c.PostForm("pvc_name")
	localPath := c.PostForm("local_path")
	description := c.PostForm("description")
	storage, _ := strconv.ParseInt(c.PostForm("storage"), 10, 64)
	createTime := time.Now().Format("2006-01-02 15:04:05")

	workspace := &model.WorkspaceInfo{
		Username:    username,
		Namespace:   namespace,
		Name:        name,
		Type:        _type,
		PvcName:     pvcName,
		LocalPath:   localPath,
		Description: description,
		CreateTime:  createTime,
		Storage:     storage,
		Status:      "Created",
	}

	if err := wc.objBackend.CreateWorkspace(workspace); err != nil {
		Error(c, fmt.Sprintf("failed to create workspace, err: %s", err))
		return
	}
	// create the data source for the workspace
	dataSource := model.DataSource{
		Name:        model.WorkspacePrefix + workspace.Name,
		PvcName:     model.WorkspacePrefix + workspace.Name,
		LocalPath:   workspace.LocalPath,
		Description: fmt.Sprintf("storage for workspace %s", workspace.Name),
		CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
		UserId:      "kubedl-system",
		Username:    "kubedl-system",
		Namespace:   workspace.Name,
	}

	err := wc.dataSourceHandler.PostDataSourceToConfigMap(dataSource)
	if err != nil {
		Error(c, fmt.Sprintf("failed to create data source config, err: %s", err))
		return
	}
	klog.Infof("create workspace %s ", workspace.Name)
	utils.Succeed(c, nil)
}

func (wc *WorkspaceAPIsController) DeleteWorkspace(c *gin.Context) {
	name := c.Param("name")
	klog.Infof("post /workspace/delete with parameters: name=%s", name)
	err := wc.objBackend.DeleteWorkspace(name)
	if err != nil {
		Error(c, fmt.Sprintf("failed to delete workspace, err: %s", err))
		return
	}

	err = wc.dataSourceHandler.DeleteDataSourceFromConfigMap(model.WorkspacePrefix + name)
	if err != nil {
		Error(c, fmt.Sprintf("failed to delete data source config, err: %s", err))
		return
	}

	utils.Succeed(c, nil)
}

func (wc *WorkspaceAPIsController) ListWorkspaces(c *gin.Context) {
	var (
		name, curPageNum, curPageSize string
	)

	query := backends.WorkspaceQuery{}
	if startTime := c.Query("start_time"); startTime != "" {
		t, err := time.Parse(time.RFC3339, startTime)
		if err != nil {
			Error(c, fmt.Sprintf("failed to parse start time[create_time=%s], err=%s", startTime, err))
			return
		}
		query.StartTime = t
	} else {
		Error(c, "start_time should not be empty")
		return
	}

	if name = c.Query("name"); name != "" {
		query.Name = name
	}

	if curPageNum = c.Query("current_page"); curPageNum != "" {
		pageNum, err := strconv.Atoi(curPageNum)
		if err != nil {
			Error(c, fmt.Sprintf("failed to parse url parameter[current_page=%s], err=%s", curPageNum, err))
			return
		}
		if query.Pagination == nil {
			query.Pagination = &backends.QueryPagination{}
		}
		query.Pagination.PageNum = pageNum
	}
	if curPageSize = c.Query("page_size"); curPageSize != "" {
		pageSize, err := strconv.Atoi(curPageSize)
		if err != nil {
			Error(c, fmt.Sprintf("failed to parse url parameter[page_size=%s], err=%s", curPageSize, err))
			return
		}
		if query.Pagination == nil {
			query.Pagination = &backends.QueryPagination{}
		}
		query.Pagination.PageSize = pageSize
	}
	klog.Infof("get /workspace/list with parameters: name=%s, pageNum=%s, pageSize=%s",
		name, curPageNum, curPageSize)

	workspaceInfos, err := wc.objBackend.ListWorkspaces(&query)
	if err != nil {
		Error(c, fmt.Sprintf("failed to list workspaces from backend, err=%v", err))
		return
	}
	utils.Succeed(c, map[string]interface{}{
		"workspaceInfos": workspaceInfos,
		"total":          query.Pagination.Count,
	})
}

func (wc *WorkspaceAPIsController) GetWorkspaceDetail(context *gin.Context) {
}
