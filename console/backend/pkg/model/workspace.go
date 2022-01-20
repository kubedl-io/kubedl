package model

const WorkspacePrefix = "workspace-"
const WorkspaceKubeDLLabel = "kubedl.io/workspace-name"

// WorkspaceInfo is the object returned in http call
type WorkspaceInfo struct {
	Username string `json:"username"`

	Namespace string `json:"namespace"`

	CPU int64 `json:"cpu"`

	Memory int64 `json:"memory"`

	GPU int64 `json:"gpu"`

	Storage int64 `json:"storage"`

	Name string `json:"name"`

	Type string `json:"type"`

	PvcName string `json:"pvc_name"`

	// Created
	// Ready: pvc bound
	Status string `json:"status"`

	LocalPath string `json:"local_path"`

	Description string `json:"description"`

	CreateTime string `json:"create_time"`

	UpdateTime string `json:"update_time"`

	DurationTime string `json:"duration_time"`
}
