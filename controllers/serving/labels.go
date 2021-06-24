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

package serving

import (
	"github.com/alibaba/kubedl/apis/serving/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func insertLabelsForPredictor(labels map[string]string, inf *v1alpha1.Inference, predictor *v1alpha1.PredictorSpec) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[v1.LabelInferenceName] = inf.Name
	labels[v1.LabelPredictorName] = predictor.Name
	labels[v1.LabelModelVersion] = predictor.ModelVersion
	return labels
}
