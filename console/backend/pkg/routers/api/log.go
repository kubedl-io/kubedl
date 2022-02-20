package api

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
)

func NewLogsAPIsController(logHandler *handlers.LogHandler) *logsAPIsController {
	return &logsAPIsController{logHandler: logHandler}
}

type logsAPIsController struct {
	logHandler *handlers.LogHandler
	lock       sync.Mutex
}

func (lc *logsAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	logAPIs := routes.Group("/log")
	logAPIs.GET("/logs/:namespace/:podName", lc.GetPodLogs)
	logAPIs.GET("/download/:namespace/:podName", lc.DownloadPodLogs)

	eventAPIs := routes.Group("/event")
	eventAPIs.GET("/events/:namespace/:objName", lc.GetEvents)
}

func (lc *logsAPIsController) GetPodLogs(c *gin.Context) {
	namespace := utils.Param(c, "namespace")
	podName := utils.Param(c, "podName")
	uid := utils.Query(c, "uid")
	from := utils.Query(c, "fromTime")
	to := utils.Query(c, "toTime")

	// Assert critical parameters are not empty.
	if namespace == "" || podName == "" || from == "" {
		utils.Failed(c, "(namespace, podName, fromTime) should not be empty")
		return
	}

	// Transform fromTime and toTime time values.
	fromTime, toTime, err := utils.TimeTransform(from, to)
	if err != nil {
		Error(c, fmt.Sprintf("failed to transform fromTime and toTime values, fromTime=%s, toTime=%s, err: %v", from, to, err))
		return
	}
	if toTime.IsZero() {
		toTime = time.Now()
	}

	logs, err := lc.logHandler.GetLogs(namespace, podName, uid, fromTime, toTime)
	if err != nil {
		Error(c, fmt.Sprintf("failed to get logs of pod %s/%s, err: %v", namespace, podName, err))
	} else {
		utils.Succeed(c, logs)
	}
}

func (lc *logsAPIsController) DownloadPodLogs(c *gin.Context) {
	namespace := utils.Param(c, "namespace")
	podName := utils.Param(c, "podName")
	uid := utils.Query(c, "uid")
	from := utils.Query(c, "fromTime")
	to := utils.Query(c, "toTime")
	// Assert critical parameters are not empty.
	if namespace == "" || podName == "" || from == "" {
		utils.Failed(c, "(namespace, podName, fromTime) should not be empty")
		return
	}

	// Transform fromTime and toTime time values.
	fromTime, toTime, err := utils.TimeTransform(from, to)
	if err != nil {
		Error(c, fmt.Sprintf("failed to transform fromTime and toTime values, fromTime=%s, toTime=%s, err: %v", from, to, err))
		return
	}
	if toTime.IsZero() {
		toTime = time.Now()
	}

	lc.lock.Lock()
	defer lc.lock.Unlock()

	// Get logs and response by a plain text-content file.
	logBytes, err := lc.logHandler.DownloadLogs(namespace, podName, uid, fromTime, toTime)
	if err != nil {
		Error(c, fmt.Sprintf("failed to download log content, pod: %s/%s, err: %v", namespace, podName, err))
		return
	} else {
		logCtx := bytes.NewReader(logBytes)
		contentLen := len(logBytes)
		contentType := "text/plain"
		extraHeaders := map[string]string{
			"Content-Disposition": fmt.Sprintf(`attachment; filename="%s_%s_%s_%d.log"`,
				namespace, podName, uid, time.Now().Unix()),
		}
		c.DataFromReader(http.StatusOK, int64(contentLen), contentType, logCtx, extraHeaders)
	}
}

func (lc *logsAPIsController) GetEvents(c *gin.Context) {
	namespace := utils.Param(c, "namespace")
	objName := utils.Param(c, "objName")
	uid := utils.Query(c, "uid")
	from := utils.Query(c, "fromTime")
	to := utils.Query(c, "toTime")

	if namespace == "" || objName == "" || from == "" {
		utils.Failed(c, "(namespace, objName, fromTime) should not be empty")
		return
	}

	fromTime, toTime, err := utils.TimeTransform(from, to)
	if err != nil {
		Error(c, fmt.Sprintf("failed to transform fromTime and toTime values, fromTime=%s, toTime=%s, err: %v", from, to, err))
		return
	}
	if toTime.IsZero() {
		toTime = time.Now()
	}

	events, err := lc.logHandler.GetEvents(namespace, objName, uid, fromTime, toTime)
	if err != nil {
		Error(c, fmt.Sprintf("failed to get job events, obj: %s/%s, err: %v", namespace, objName, err))
	} else {
		utils.Succeed(c, events)
	}
}
