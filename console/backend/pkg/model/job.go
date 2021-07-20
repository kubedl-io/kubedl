package model

import (
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	corev1 "k8s.io/api/core/v1"
)

type JobResource struct {
	TotalCPU    int64 `json:"totalCPU"`
	TotalMemory int64 `json:"totalMemory"`
	TotalGPU    int64 `json:"totalGPU"`
}

// JobInfo job meta info
type JobInfo struct {
	// Id: unique id
	Id string `json:"id"`
	// Name: job name
	Name string `json:"name"`
	// JobType: type
	JobType string `json:"jobType"`
	// Enable Tensorboard or not
	EnableTensorboard bool `json:"enableTensorboard"`
	// status of job
	JobStatus v1.JobConditionType `json:"jobStatus"`
	// Namespace
	Namespace string `json:"namespace"`
	//
	CreateTime string `json:"createTime"`
	//
	EndTime string `json:"endTime"`
	//
	DurationTime string `json:"durationTime"`
	//
	DeployRegion string `json:"deployRegion"`
	//
	ExitedEvents []string `json:"exitedEvents"`
	//
	Specs []Spec `json:"specs"`
	//
	SpecsReplicaStatuses map[string]*SpecReplicaStatus `json:"specsReplicaStatuses"`

	JobConfig string `json:"jobConfig,omitempty"`

	JobUserName string `json:"jobUserName,omitempty"`
	//JobResource JobResource `json:"jobResource,omitempty"`
}

// Spec pods spec of job
type Spec struct {
	Name  string `json:"name"`
	PodId string `json:"podId"`
	//
	ReplicaType string `json:"replicaType"`
	//
	ContainerIp string `json:"containerIp"`
	//
	ContainerId string `json:"containerId"`
	//
	HostIp string `json:"hostIp"`
	//
	Status corev1.PodPhase `json:"jobStatus"`
	//
	CreateTime string `json:"createTime"`
	//
	StartTime string `json:"startTime"`
	//
	EndTime string `json:"endTime"`
	//
	DurationTime string `json:"durationTime"`
	// Reason of failed
	Reason string `json:"reason,omitempty"`
	// Message of failed
	Message string `json:"message,omitempty"`
	// Reason + Message
	Remark string `json:"remark,omitempty"`
}

// SpecReplicaStatus status of pods
type SpecReplicaStatus struct {
	// The number of actively running pods.
	Active int32 `json:"active"`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed"`

	// The number of pods which reached phase Stopped.
	Stopped int32 `json:"stopped"`
}

// HistoryJobStatistic used to record history job statistic.
type HistoryJobStatistic struct {
	// UserName can be JobUserName in JobInfo
	// or "Anonymous" if JobUserName is empty.
	UserName string `json:"userName"`

	// Total job count submitted by this user.
	JobCount int32 `json:"jobCount"`

	// Job ratio submitted by this user.
	JobRatio float64 `json:"jobRatio"`
}

// JobStatistics used to record job statistics
// submitted by users in current cluster.
type JobStatistics struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`

	// Total job count submmitted from startTime to endTime.
	TotalJobCount int32 `json:"totalJobCount"`

	// Statistics of history jobs (All status), group by user.
	HistoryJobs []*HistoryJobStatistic `json:"historyJobs"`
}
