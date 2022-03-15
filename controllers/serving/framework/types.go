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

package framework

import (
	v1 "k8s.io/api/core/v1"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	"github.com/alibaba/kubedl/apis/serving/v1alpha1"
)

// Setter is an interface definition to setup specification for predictor template.
type Setter interface {
	SetSpec(template *v1.PodTemplateSpec, modelVersion *modelv1alpha1.ModelVersion, modelPath string)
}

var (
	Setters = make(map[v1alpha1.FrameworkType]Setter)
)
