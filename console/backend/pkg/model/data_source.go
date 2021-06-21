package model

type DataSource struct {
	UserId string `json:"userid"`

	Username string `json:"username"`

	Namespace string `json:"namespace"`

	Name string `json:"name"`

	Type string `json:"type"`

	PvcName string `json:"pvc_name"`

	LocalPath string `json:"local_path"`

	Description string `json:"description"`

	CreateTime string `json:"create_time"`

	UpdateTime string `json:"update_time"`
}

type DataSourceMap map[string]DataSource
