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
	"path/filepath"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	"github.com/alibaba/kubedl/apis/serving/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func init() {
	Setters[v1alpha1.FrameworkTFServing] = &tfServingSetter{}
}

const (
	TensorflowServingGRPCPort         = "9000"
	TensorflowServingRestPort         = "8080"
	EnvTensorflowServingModelName     = "MODEL_NAME"
	EnvTensorflowServingModelBasePath = "MODEL_BASE_PATH"
)

var _ Setter = &tfServingSetter{}

type tfServingSetter struct{}

func (t *tfServingSetter) SetSpec(template *corev1.PodTemplateSpec, modelVersion *modelv1alpha1.ModelVersion, modelPath string) {
	for ci := range template.Spec.Containers {
		c := &template.Spec.Containers[ci]
		c.Env = append(c.Env,
			corev1.EnvVar{Name: EnvTensorflowServingModelBasePath, Value: filepath.Dir(modelPath) /* parent dir as base path */},
			corev1.EnvVar{Name: modelv1alpha1.KubeDLModelPath, Value: modelPath},
		)
		if modelVersion != nil && modelVersion.Name != "" {
			c.Env = append(c.Env, corev1.EnvVar{Name: EnvTensorflowServingModelName, Value: modelVersion.Spec.ModelName})
		}
	}
}
