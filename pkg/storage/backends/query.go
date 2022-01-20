/*
Copyright 2020 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backends

import (
	"time"

	"github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// Query for job via http
type Query struct {
	JobID      string
	Name       string
	Namespace  string
	Type       string
	Region     string
	Status     v1.JobConditionType
	StartTime  time.Time
	EndTime    time.Time
	IsDel      *int
	Pagination *QueryPagination
}

// NotebookQuery for notebooks via http
type NotebookQuery struct {
	NotebookID string
	Name       string
	Namespace  string
	Region     string
	Status     v1alpha1.NotebookCondition
	StartTime  time.Time
	EndTime    time.Time
	IsDel      *int
	Pagination *QueryPagination
}

type WorkspaceQuery struct {
	WorkspaceID string
	Name        string
	Namespace   string
	Region      string
	StartTime   time.Time
	EndTime     time.Time
	IsDel       *int
	Pagination  *QueryPagination
}

type QueryPagination struct {
	PageNum  int
	PageSize int
	Count    int
}
