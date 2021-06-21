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

// JobInfo job的基础模型
// 一期简单定义- 平铺
type JobInfo struct {
	// 任务标识
	Id string `json:"id"`
	// 任务名称
	Name string `json:"name"`
	// 任务类型
	JobType string `json:"jobType"`
	// 是否启用了TensorBoard
	EnableTensorboard bool `json:"enableTensorboard"`
	// 任务状态
	JobStatus v1.JobConditionType `json:"jobStatus"`
	// 命名空间
	Namespace string `json:"namespace"`
	// 创建时间
	CreateTime string `json:"createTime"`
	// 结束时间
	EndTime string `json:"endTime"`
	// 执行时长
	DurationTime string `json:"durationTime"`
	// 部署域
	DeployRegion string `json:"deployRegion"`
	// 退出事件列表
	ExitedEvents []string `json:"exitedEvents"`
	// 任务规格列表
	Specs []Spec `json:"specs"`
	//任务规格类型状态统计
	SpecsReplicaStatuses map[string]*SpecReplicaStatus `json:"specsReplicaStatuses"`

	JobConfig   string `json:"jobConfig,omitempty"`
	JobUserID   string `json:"jobUserId,omitempty"`
	JobUserName string `json:"jobUserName,omitempty"`
	//JobResource JobResource `json:"jobResource,omitempty"`
}

// Spec 任务规格模型
type Spec struct {
	// 名称
	Name  string `json:"name"`
	PodId string `json:"podId"`
	//任务类型
	ReplicaType string `json:"replicaType"`
	// 容器IP
	ContainerIp string `json:"containerIp"`
	// 容器ID
	ContainerId string `json:"containerId"`
	// 宿主机IP
	HostIp string `json:"hostIp"`
	// 状态
	Status corev1.PodPhase `json:"jobStatus"`
	// 创建时间
	CreateTime string `json:"createTime"`
	// 开始时间
	StartTime string `json:"startTime"`
	// 结束时间
	EndTime string `json:"endTime"`
	// 执行时长
	DurationTime string `json:"durationTime"`
	// 原因，目前只有失败情况下才有
	Reason string `json:"reason,omitempty"`
	// 描述，目前只有失败情况下才有
	Message string `json:"message,omitempty"`
	// 原因+描述，目前只有失败情况下才有
	Remark string `json:"remark,omitempty"`
}

// SpecReplicaStatus 规格状态
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
	UserID string `json:"userID"`
	// UserName can be JobUserName in JobInfo
	// or JobUserID iff JobUserName is empty in JobInfo
	// or "Anonymous" if JobUserName and JobUserID are both empty.
	UserName string `json:"userName"`

	// Total job count submmitted by this user.
	JobCount int32 `json:"jobCount"`

	// Job ratio submmited by this user.
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
