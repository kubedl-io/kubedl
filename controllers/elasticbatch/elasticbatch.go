// Copyright 2022 The Alibaba Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package controller provides a Kubernetes controller for a ElasticBatchJob resource.
package elasticbatch

import (
	"context"
	"encoding/json"
	"strconv"

	inference "github.com/alibaba/kubedl/apis/inference/v1alpha1"
)

type ElasticBatchConfig struct {
	Task        TaskSpec `json:"task"`
	Environment string   `json:"environment"`
}

// TaskSpec is the specification for a task (PS or worker) of the ElasticBatchJob.
type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// Generate the environment variable ElASTICBATCH_CONFIG
//
//	{
//	    "task": {
//	        "type": "worker",
//	        "index": 1
//	        },
//	}
func genElasticBatchConfigJSONStr(ctx context.Context, elasticbatchjob *inference.ElasticBatchJob, rtype, index string) (string, error) {
	// Configure the ElASTICBATCH_CONFIG environment variable.
	i, err := strconv.ParseInt(index, 0, 32)
	if err != nil {
		return "", err
	}

	elasticBatchConfig := ElasticBatchConfig{
		Task: TaskSpec{
			Type:  rtype,
			Index: int(i),
		},
		Environment: "cloud",
	}

	elasticBatchConfigJSONStr, err := json.Marshal(elasticBatchConfig)
	if err != nil {
		return "", err
	}

	return string(elasticBatchConfigJSONStr), nil
}
