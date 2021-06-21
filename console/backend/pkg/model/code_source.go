package model

type CodeSource struct {
	UserId string `json:"userid"`

	Username string `json:"username"`

	Name string `json:"name"`

	Type string `json:"type"`

	CodePath string `json:"code_path"`

	DefaultBranch string `json:"default_branch"`

	LocalPath string `json:"local_path"`

	Description string `json:"description"`

	CreateTime string `json:"create_time"`

	UpdateTime string `json:"update_time"`
}

type CodeSourceMap map[string]CodeSource
