package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog"

	"github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
	"github.com/alibaba/kubedl/pkg/storage/backends"
)

func NewNotebookAPIsController(notebookHandler *handlers.NotebookHandler) *notebookAPIsController {
	return &notebookAPIsController{notebookHandler: notebookHandler}

}

type notebookAPIsController struct {
	notebookHandler *handlers.NotebookHandler
}

func (nc *notebookAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	notebookAPI := routes.Group("/notebook")
	notebookAPI.GET("/list", nc.ListNotebooks)
	notebookAPI.POST("/submit", nc.SubmitNotebook)
	notebookAPI.DELETE("/:namespace/:name", nc.DeleteNotebook)
	notebookAPI.GET("/yaml/:namespace/:name", nc.GetYamlData)
	notebookAPI.GET("/json/:namespace/:name", nc.GetJsonData)
}

func (nc *notebookAPIsController) SubmitNotebook(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		Error(c, "failed to get raw posted data from request")
		return
	}
	if err = nc.notebookHandler.SubmitNotebook(data); err != nil {
		Error(c, fmt.Sprintf("failed to submit notebook, err: %s", err))
		return
	}
	utils.Succeed(c, nil)
}

func (nc *notebookAPIsController) DeleteNotebook(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	id := c.Query("id")
	klog.Infof("post /notebook/delete with parameters: namespace=%s, name=%s", namespace, name)
	err := nc.notebookHandler.DeleteNotebookFromBackend(namespace, name, id, "")
	if err != nil {
		Error(c, fmt.Sprintf("failed to delete notebook, err: %s", err))
	} else {
		utils.Succeed(c, nil)
	}
}

func (nc *notebookAPIsController) ListNotebooks(c *gin.Context) {
	var (
		ns, name, status, curPageNum, curPageSize string
	)

	query := backends.NotebookQuery{}
	if startTime := c.Query("start_time"); startTime != "" {
		t, err := time.Parse(time.RFC3339, startTime)
		if err != nil {
			Error(c, fmt.Sprintf("failed to parse start time[start_time=%s], err=%s", startTime, err))
			return
		}
		query.StartTime = t
	} else {
		Error(c, "start_time should not be empty")
		return
	}
	if endTime := c.Query("end_time"); endTime != "" {
		t, err := time.Parse(time.RFC3339, endTime)
		if err != nil {
			Error(c, fmt.Sprintf("failed to parse end time[end_time=%s], err=%s", endTime, err))
			return
		}
		query.EndTime = t
	} else {
		query.EndTime = time.Now()
	}
	if ns = c.Query("namespace"); ns != "" {
		query.Namespace = ns
	}
	if name = c.Query("name"); name != "" {
		query.Name = name
	}
	if status = c.Query("status"); status != "" {
		query.Status = v1alpha1.NotebookCondition(status)
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
	klog.Infof("get /notebook/list with parameters: namespace=%s, name=%s, status=%s, pageNum=%s, pageSize=%s",
		ns, name, status, curPageNum, curPageSize)

	notebookInfos, err := nc.notebookHandler.ListNotebooksFromBackend(&query)
	if err != nil {
		Error(c, fmt.Sprintf("failed to list notebooks from backend, err=%v", err))
		return
	}
	utils.Succeed(c, map[string]interface{}{
		"notebookInfos": notebookInfos,
		"total":         query.Pagination.Count,
	})
}

func (nc *notebookAPIsController) GetYamlData(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	kind := c.Query("kind")

	data, err := nc.notebookHandler.GetYamlData(namespace, name, kind)
	if err != nil {
		Error(c, err.Error())
		return
	}
	utils.Succeed(c, string(data))
}

func (nc *notebookAPIsController) GetJsonData(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	data, err := nc.notebookHandler.GetJsonData(namespace, name, "Notebook")
	if err != nil {
		Error(c, err.Error())
		return
	}

	utils.Succeed(c, string(data))
}
