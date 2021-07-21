package model

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/alibaba/kubedl/pkg/storage/dmo/converters"

	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/util"
)

const (
	JobInfoTimeFormat = "2006-01-02 15:04:05"
)

func ConvertDMOJobToJobInfo(dmoJob *dmo.Job) JobInfo {
	jobInfo := JobInfo{
		Id:        dmoJob.JobID,
		Name:      dmoJob.Name,
		JobType:   dmoJob.Kind,
		JobStatus: dmoJob.Status,
		Namespace: dmoJob.Namespace,
	}
	if dmoJob.DeployRegion != nil {
		jobInfo.DeployRegion = *dmoJob.DeployRegion
	}

	if !dmoJob.GmtJobSubmitted.IsZero() {
		jobInfo.CreateTime = dmoJob.GmtJobSubmitted.Local().Format(JobInfoTimeFormat)
	}
	if !util.Time(dmoJob.GmtJobFinished).IsZero() {
		jobInfo.EndTime = dmoJob.GmtJobFinished.Local().Format(JobInfoTimeFormat)
	}
	if !dmoJob.GmtJobSubmitted.IsZero() && !util.Time(dmoJob.GmtJobFinished).IsZero() {
		jobInfo.DurationTime = GetTimeDiffer(dmoJob.GmtJobSubmitted, *dmoJob.GmtJobFinished)
	}
	if dmoJob.Remark != nil {
		for _, remark := range strings.Split(*dmoJob.Remark, ",") {
			if strings.TrimSpace(remark) == converters.RemarkEnableTensorBoard {
				jobInfo.EnableTensorboard = true
				break
			}
		}
	}
	jobInfo.JobUserName = *dmoJob.Owner

	/*
		jobResource, err := calculateJobResources(jobInfo.Resources)
		if err != nil {
			klog.Errorf("computeJobResources failed, err: %v", err)
		}

		jobInfo.JobResource = JobResource{
			TotalCPU:    jobResource.Cpu().MilliValue(),
			TotalMemory: jobResource.Memory().Value(),
			TotalGPU:    resource_utils.GetGpuResource(jobResource).MilliValue(),
		}
	*/

	return jobInfo
}

func ConvertDMOPodToJobSpec(pod *dmo.Pod) Spec {
	spec := Spec{
		Name:        pod.Name,
		PodId:       pod.PodID,
		ReplicaType: pod.ReplicaType,
		Status:      pod.Status,
	}
	if pod.PodIP != nil {
		spec.ContainerIp = *pod.PodIP
	}
	if pod.HostIP != nil {
		spec.HostIp = *pod.HostIP
	}
	if pod.Remark != nil {
		spec.Remark = *pod.Remark
	}
	if !pod.GmtCreated.IsZero() {
		spec.CreateTime = pod.GmtCreated.Local().Format(JobInfoTimeFormat)
	}
	if pod.GmtStarted != nil && !pod.GmtStarted.IsZero() {
		spec.StartTime = pod.GmtStarted.Local().Format(JobInfoTimeFormat)
	}
	if !util.Time(pod.GmtFinished).IsZero() {
		spec.EndTime = pod.GmtFinished.Local().Format(JobInfoTimeFormat)
	}
	if !pod.GmtCreated.IsZero() && !util.Time(pod.GmtFinished).IsZero() {
		spec.DurationTime = GetTimeDiffer(pod.GmtCreated, *pod.GmtFinished)
	}
	return spec
}

// GetTimeDiffer computes time differ duration between 2 time values, formated as
// 2h2m2s.
func GetTimeDiffer(startTime time.Time, endTime time.Time) (differ string) {
	seconds := endTime.Sub(startTime).Seconds()
	var buffer bytes.Buffer
	hours := math.Floor(seconds / 3600)
	if hours > 0 {
		buffer.WriteString(strconv.FormatFloat(hours, 'g', -1, 64))
		buffer.WriteString("h")
		seconds = seconds - 3600*hours
	}
	minutes := math.Floor(seconds / 60)
	if minutes > 0 {
		buffer.WriteString(strconv.FormatFloat(minutes, 'g', -1, 64))
		buffer.WriteString("m")
		seconds = seconds - 60*minutes
	}
	buffer.WriteString(strconv.FormatFloat(seconds, 'g', -1, 64))
	buffer.WriteString("s")
	return buffer.String()
}
