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

package v1alpha1

import v1 "k8s.io/api/core/v1"

// Default constants values for inference service.
const (
	DefaultPredictorContainerName         = "predictor"
	DefaultPredictorContainerHttpPortName = "predictor-http-port"
	DefaultPredictorContainerGrpcPortName = "predictor-grpc-port"
	DefaultPredictorContainerHttpPort     = 8080
	DefaultPredictorContainerGrpcPort     = 9000
	DefaultModelMountPath                 = "/mnt/models"
)

func SetDefaults_Inference(in *Inference) {
	for pi := range in.Spec.Predictors {
		setDefaults_predictor(&in.Spec.Predictors[pi])
	}
}

func setDefaults_predictor(predictor *PredictorSpec) {
	if predictor.Replicas == nil {
		replicas := int32(1)
		predictor.Replicas = &replicas
	}

	index := 0
	for i, container := range predictor.Template.Spec.Containers {
		if container.Name == DefaultPredictorContainerName {
			index = i
			break
		}
	}

	hasPredictorPort := false
	for _, port := range predictor.Template.Spec.Containers[index].Ports {
		if port.Name == DefaultPredictorContainerHttpPortName {
			hasPredictorPort = true
			break
		}
	}
	if !hasPredictorPort {
		predictor.Template.Spec.Containers[index].Ports = append(predictor.Template.Spec.Containers[index].Ports, v1.ContainerPort{
			Name:          DefaultPredictorContainerHttpPortName,
			ContainerPort: DefaultPredictorContainerHttpPort,
		})
	}
}
