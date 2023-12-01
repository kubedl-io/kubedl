package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"

	"github.com/alibaba/kubedl/console/backend/pkg/auth"

	"github.com/gin-gonic/gin"

	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/backends"

	"k8s.io/klog"
)

func NewJobAPIsController(jobHandler *handlers.JobHandler) *jobAPIsController {
	return &jobAPIsController{jobHandler: jobHandler}
}

type jobAPIsController struct {
	jobHandler *handlers.JobHandler
}

func (jc *jobAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	jobAPI := routes.Group("/job")
	jobAPI.GET("/list", jc.ListJobs)
	jobAPI.GET("/detail", jc.GetJobDetail)
	jobAPI.GET("/yaml/:namespace/:name", jc.GetJobYamlData)
	jobAPI.GET("/json/:namespace/:name", jc.GetJobJsonData)
	jobAPI.POST("/stop", jc.StopJob)
	jobAPI.POST("/submit", jc.SubmitJob)
	jobAPI.DELETE("/:namespace/:name", jc.DeleteJob)
	jobAPI.GET("/statistics", jc.GetJobStatistics)
	jobAPI.GET("/running-jobs", jc.GetRunningJobs)

	pvcAPIs := routes.Group("/pvc")
	pvcAPIs.GET("/list", jc.ListPVC)
}

func (jc *jobAPIsController) ListJobs(c *gin.Context) {
	var (
		kind, ns, name, status, curPageNum, curPageSize string
	)

	query := backends.Query{}
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
	if kind = c.Query("kind"); kind != "" {
		query.Type = kind
	}
	if ns = c.Query("namespace"); ns != "" {
		query.Namespace = ns
	}
	if name = c.Query("name"); name != "" {
		query.Name = name
	}
	if status = c.Query("status"); status != "" {
		query.Status = v1.JobConditionType(status)
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
	klog.Infof("get /job/list with parameters: kind=%s, namespace=%s, name=%s, status=%s, pageNum=%s, pageSize=%s",
		kind, ns, name, status, curPageNum, curPageSize)
	jobInfos, err := jc.jobHandler.ListJobsFromBackend(&query)
	if err != nil {
		Error(c, fmt.Sprintf("failed to list jobs from backend, err=%v", err))
		return
	}
	utils.Succeed(c, map[string]interface{}{
		"jobInfos": jobInfos,
		"total":    query.Pagination.Count,
	})
}

func (jc *jobAPIsController) GetJobDetail(c *gin.Context) {
	var (
		jobID, deployRegion, namespace, jobName, kind string
	)

	deployRegion = c.Query("deploy_region")
	jobName = c.Query("job_name")
	namespace = c.Query("namespace")
	kind = c.Query("kind")

	if kind == "" {
		Error(c, "job kind must not be empty")
		return
	}

	if jobName != "" {
		if namespace == "" {
			Error(c, "namespace should not be empty")
			return
		}

		var (
			err                      error
			gmtStartDate, gmtEndDate time.Time
		)

		startDate := c.Query("start_date")
		if startDate == "" {
			gmtStartDate = time.Now().Add(-1 * time.Hour * 24)
		} else {
			gmtStartDate, err = time.Parse("2006-01-02", startDate)
			if err != nil {
				Error(c, fmt.Sprintf("parse start date[%s] error: %v", startDate, err))
				return
			}
		}

		endDate := c.Query("end_date")
		if endDate == "" || endDate == startDate {
			gmtEndDate = time.Now()
		} else {
			gmtEndDate, err = time.Parse("2006-01-02", startDate)
			if err != nil {
				Error(c, fmt.Sprintf("parse end date[%s] error: %v", endDate, err))
				return
			}
		}

		jobs, err := jc.jobHandler.ListJobsFromBackend(&backends.Query{
			Name:      jobName,
			Namespace: namespace,
			StartTime: gmtStartDate,
			EndTime:   gmtEndDate,
		})
		if err != nil {
			Error(c, fmt.Sprintf("failed to list job from backend, err: %v", err))
			return
		}
		if len(jobs) == 0 {
			Error(c, fmt.Sprintf("job %s/%s not found", namespace, jobName))
			return
		}
		job := jobs[0]
		jobID, deployRegion = job.Id, job.DeployRegion
	} else {
		jobID = c.Query("job_id")
	}

	currentPage, err := strconv.Atoi(c.Query("current_page"))
	if err != nil {
		Error(c, fmt.Sprintf("invalid current page [%s], err: %v", c.Query("current_page"), err))
		return
	}
	if currentPage <= 0 {
		currentPage = 1
	}

	pageSize, err := strconv.Atoi(c.Query("page_size"))
	if err != nil {
		Error(c, fmt.Sprintf("invalid page size [%s], err: %v", c.Query("page_size"), err))
		return
	}

	klog.Infof("get /job/detail with parameters: kind=%s, namespace=%s, name=%s, id=%s, currentPage=%d, deployRegion=%s",
		kind, namespace, jobName, jobID, currentPage, deployRegion)
	jobInfo, err := jc.jobHandler.GetDetailedJobFromBackend(namespace, jobName, jobID, kind, deployRegion)
	if err != nil {
		Error(c, fmt.Sprintf("failed to get detailed job from backend, namespace=%s, name=%s, id=%s, kind=%s, err: %v",
			namespace, jobName, jobID, kind, err))
		return
	}

	replicaType := c.Query("replica_type")
	if replicaType == "" {
		replicaType = "ALL"
	}
	status := c.Query("status")
	if status == "" {
		status = "ALL"
	}

	// Filter jobs by replica type and job status
	jobInfo.Specs = jobFilter(replicaType, status, jobInfo.Specs)
	originTotal := len(jobInfo.Specs)
	startIdx := (currentPage - 1) * pageSize
	if startIdx > len(jobInfo.Specs) {
		utils.Failed(c, "current page out of index")
		return
	}
	endIdx := currentPage * pageSize
	if endIdx > len(jobInfo.Specs) {
		endIdx = len(jobInfo.Specs)
	}
	jobInfo.Specs = jobInfo.Specs[startIdx:endIdx]
	utils.Succeed(c, map[string]interface{}{
		"jobInfo": jobInfo,
		"total":   originTotal,
	})
}

func (jc *jobAPIsController) StopJob(c *gin.Context) {
	name := c.Param("name")
	namespace := c.Param("namespace")
	region := c.Param("deployRegion")
	kind := c.Query("kind")
	klog.Infof("post /job/stop with parameters: kind=%s, namespace=%s, name=%s", region, namespace, name)
	err := jc.jobHandler.StopJobFromBackend(namespace, name, "", kind, region)
	if err != nil {
		Error(c, fmt.Sprintf("failed to stop job, err: %s", err))
	} else {
		utils.Succeed(c, nil)
	}
}

func (jc *jobAPIsController) DeleteJob(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	id := c.Query("id")
	kind := c.Query("kind")
	klog.Infof("post /job/delete with parameters: kind=%s, namespace=%s, name=%s", kind, namespace, name)
	err := jc.jobHandler.DeleteJobFromBackend(namespace, name, id, kind, "")
	if err != nil {
		Error(c, fmt.Sprintf("failed to delete job, err: %s", err))
	} else {
		utils.Succeed(c, nil)
	}
}

func (jc *jobAPIsController) ListPVC(c *gin.Context) {
	namespace := c.Query("namespace")
	if pvc, err := jc.jobHandler.ListPVC(namespace); err != nil {
		Error(c, fmt.Sprintf("failed to list pvc, error: %v", err))
		return
	} else {
		utils.Succeed(c, pvc)
	}
}

func (jc *jobAPIsController) SubmitJob(c *gin.Context) {
	kind := c.Query("kind")
	if kind == "" {
		Error(c, "job kind is empty")
		return
	}
	data, err := c.GetRawData()
	if err != nil {
		Error(c, "failed to get raw posted data from request")
		return
	}
	if err = jc.jobHandler.SubmitJobWithKind(data, kind); err != nil {
		Error(c, fmt.Sprintf("failed to submit job, err: %s", err))
		return
	}
	utils.Succeed(c, nil)
}

func (jc *jobAPIsController) GetJobYamlData(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	kind := c.Query("kind")

	data, err := jc.jobHandler.GetJobYamlData(namespace, name, kind)
	if err != nil {
		Error(c, err.Error())
		return
	}
	utils.Succeed(c, string(data))
}

func (jc *jobAPIsController) GetJobJsonData(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	kind := c.Query("kind")

	data, err := jc.jobHandler.GetJobJsonData(namespace, name, kind)
	if err != nil {
		Error(c, err.Error())
		return
	}

	utils.Succeed(c, string(data))
}

func (jc *jobAPIsController) GetJobStatistics(c *gin.Context) {
	query := backends.Query{}
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
	klog.Infof("get /job/statistics with parameters: start_time:%s, end_time:%s",
		query.StartTime.Format(time.RFC3339), query.EndTime.Format(time.RFC3339))
	jobStatistics, err := jc.jobHandler.GetJobStatisticsFromBackend(&query)
	if err != nil {
		Error(c, fmt.Sprintf("failed to get jobs statistics from backend, err=%v", err))
		return
	}
	utils.Succeed(c, map[string]interface{}{
		"jobStatistics": jobStatistics,
	})
}

func (jc *jobAPIsController) GetRunningJobs(c *gin.Context) {
	query := backends.Query{}
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

	query.Status = v1.JobRunning

	limit := int(-1)
	if limitPara := c.Query("limit"); limitPara != "" {
		var err error
		limit, err = strconv.Atoi(limitPara)
		if err != nil {
			Error(c, fmt.Sprintf("failed to parse limit[limit=%s], err=%s", limitPara, err))
			return
		}
	}

	klog.Infof("get /job/running-jobs with parameters: start_time:%s, end_time:%s",
		query.StartTime.Format(time.RFC3339), query.EndTime.Format(time.RFC3339))
	runningJobs, err := jc.jobHandler.GetRunningJobsFromBackend(&query)
	if err != nil {
		Error(c, fmt.Sprintf("failed to get running jobs from backend, err=%v", err))
		return
	}

	if limit != -1 && limit <= len(runningJobs) {
		runningJobs = runningJobs[0:limit]
	}
	utils.Succeed(c, map[string]interface{}{
		"runningJobs": runningJobs,
	})
}

// Error logs error info and handles a Failed response
func Error(c *gin.Context, msg string) {
	formattedMsg := msg
	session := sessions.Default(c)
	userID := session.Get(auth.SessionKeyLoginID)
	if userID != nil && userID.(string) != "" {
		formattedMsg = fmt.Sprintf("Error: %s, UserID: %s", msg, userID.(string))
	}
	klog.Error(formattedMsg)
	utils.Failed(c, msg)
}

func jobFilter(replica, status string, jobs []model.Spec) []model.Spec {
	if replica == "ALL" && status == "ALL" {
		return jobs
	}
	filtered := make([]model.Spec, 0)
	for i := range jobs {
		if (replica == "ALL" || replica == strings.ToUpper(jobs[i].ReplicaType)) &&
			(status == "ALL" || status == strings.ToUpper(string(jobs[i].Status))) {
			filtered = append(filtered, jobs[i])
		}
	}
	return filtered
}
