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
	"fmt"

	"github.com/alibaba/kubedl/apis/serving/v1alpha1"
)

// genPredictorName generate predictor name formatted as {inference name}-{predictor name}.
func genPredictorName(inf *v1alpha1.Inference, predictor *v1alpha1.PredictorSpec) string {
	return fmt.Sprintf("%s-%s", inf.Name, predictor.Name)
}

func svcHostForInference(inf *v1alpha1.Inference) string {
	return fmt.Sprintf("%s.%s", inf.Name, inf.Namespace)
}

func svcHostForPredictor(inf *v1alpha1.Inference, predictor *v1alpha1.PredictorSpec) string {
	return fmt.Sprintf("%s.%s.svc", genPredictorName(inf, predictor), inf.Namespace)
}
