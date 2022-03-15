/*
Copyright 2021 The Alibaba Authors.

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

package runtime

import (
	"encoding/json"

	"github.com/tidwall/gjson"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusTraitor traits workload status field from object scheme.
func StatusTraitor(obj metav1.Object) (apiv1.JobStatus, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return apiv1.JobStatus{}, err
	}
	status := apiv1.JobStatus{}
	if err = json.Unmarshal([]byte(gjson.GetBytes(bytes, "status").String()), &status); err != nil {
		return apiv1.JobStatus{}, err
	}
	return status, nil
}
