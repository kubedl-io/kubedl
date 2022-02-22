package model

import (
	corev1 "k8s.io/api/core/v1"
)

type NotebookResource struct {
	TotalCPU    int64 `json:"totalCPU"`
	TotalMemory int64 `json:"totalMemory"`
	TotalGPU    int64 `json:"totalGPU"`
}

// NotebookInfo for the http request
type NotebookInfo struct {
	// Id: unique id
	Id string `json:"id"`
	// Name: Notebook name
	Name string `json:"name"`

	// status of Notebook
	NotebookStatus string `json:"notebookStatus"`

	// url to the notebook
	Url string `json:"url"`

	// Namespace where the notebook instance is
	Namespace string `json:"namespace"`

	CreateTime string `json:"createTime"`

	EndTime string `json:"endTime"`

	DurationTime string `json:"durationTime"`

	DeployRegion string `json:"deployRegion"`

	NotebookConfig string `json:"notebookConfig,omitempty"`

	UserName string `json:"userName,omitempty"`

	NotebookResource NotebookResource `json:"notebookResource,omitempty"`
}

// PodSpec for Notebook
type PodSpec struct {
	Name  string `json:"name"`
	PodId string `json:"podId"`

	//
	ContainerIp string `json:"containerIp"`
	//
	ContainerId string `json:"containerId"`
	//
	HostIp string `json:"hostIp"`
	//
	Status corev1.PodPhase `json:"podStatus"`
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

// HistoryNotebookStatistic used to record history Notebook statistic.
type HistoryNotebookStatistic struct {
	// UserName can be NotebookUserName in NotebookInfo
	// or "Anonymous" if NotebookUserName is empty.
	UserName string `json:"userName"`

	// Total Notebook count submitted by this user.
	NotebookCount int32 `json:"NotebookCount"`

	// Notebook ratio submitted by this user.
	NotebookRatio float64 `json:"NotebookRatio"`
}

// NotebookStatistics used to record Notebook statistics
// submitted by users in current cluster.
type NotebookStatistics struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`

	// Total Notebook count submmitted from startTime to endTime.
	TotalNotebookCount int32 `json:"totalNotebookCount"`

	// Statistics of history Notebooks (All status), group by user.
	HistoryNotebooks []*HistoryNotebookStatistic `json:"historyNotebooks"`
}
