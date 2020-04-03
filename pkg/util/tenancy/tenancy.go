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

package tenancy

import (
	"encoding/json"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BasicTenancy struct {
	Tenant string `json:"tenant"`
	User   string `json:"user"`
}

type Tenancy struct {
	BasicTenancy `json:",inline"`
	IDC          string `json:"idc,omitempty"`
	Region       string `json:"region,omitempty"`
}

func GetTenancy(metaObj metav1.Object) (*Tenancy, error) {
	if tenancy, ok := metaObj.GetAnnotations()[v1.AnnotationTenancyInfo]; ok {
		t := Tenancy{}
		err := json.Unmarshal([]byte(tenancy), &t)
		return &t, err
	}
	return nil, nil
}
