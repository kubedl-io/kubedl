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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +k8s:defaulter-gen=true

// Inference is the Schema for the inference API.
type Inference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InferenceSpec   `json:"spec,omitempty"`
	Status InferenceStatus `json:"status,omitempty"`
}

// InferenceSpec defines the desired state of Inference
type InferenceSpec struct {
	// Framework specify the backend inference framework.
	Framework FrameworkType `json:"framework"`

	// Predictors defines a list of predictor specifications and serving-related strategies.
	Predictors []PredictorSpec `json:"predictors"`

	// TODO: support multi-stage prediction description.
}

// InferenceStatus defines the observed state of Inference
type InferenceStatus struct {
	// InferenceEndpoints exposes available serving service endpoint.
	InferenceEndpoint string `json:"inferenceEndpoint,omitempty"`
	// PredictorStatuses exposes current observed status for each predictor.
	PredictorStatuses []PredictorStatus `json:"predictorStatuses,omitempty"`
}

type PredictorSpec struct {
	// Name indicates the predictor name.
	Name string `json:"name"`

	// ModelVersion specifies the name of target model-with-version to be loaded
	// in serving service, ModelVersion object has to be created before serving
	// service deployed.
	ModelVersion string `json:"modelVersion,omitempty"`

	// ModelPath is the loaded model filepath in container.
	ModelPath *string `json:"modelPath,omitempty"`

	// Replicas specify the expected predictor replicas.
	Replicas *int32 `json:"replicas,omitempty"`

	// TrafficWeight defines the traffic split weights across multiple predictors
	TrafficWeight *int32 `json:"trafficWeight,omitempty"`

	// Template describes a template of predictor pod with its properties.
	Template corev1.PodTemplateSpec `json:"template"`

	// AutoScale describes auto-scaling strategy for predictor.
	AutoScale *AutoScaleStrategy `json:"autoScale,omitempty"`

	// Batching specify batching strategy to control how to forward a batch of
	// aggregated inference requests to backend service.
	Batching *BatchingStrategy `json:"batching,omitempty"`
}

type PredictorStatus struct {
	// Name is the name of current predictor.
	Name string `json:"name"`
	// Replicas is the expected replicas of current predictor.
	Replicas int32 `json:"replicas"`
	// ReadyReplicas is the ready replicas of current predictor.
	ReadyReplicas int32 `json:"readyReplicas"`
	// TrafficPercent is the traffic distribution routed to this service.
	TrafficPercent *int32 `json:"trafficPercent,omitempty"`
	// Endpoint is the available endpoints of current predictor.
	Endpoint string `json:"endpoint"`
}

type BatchingStrategy struct {
	// BatchSize represents batch size of requests to be forwarded to backend
	// service.
	BatchSize int32 `json:"batchSize"`
	// TimeoutSeconds represents max waiting duration to aggregate batch requests.
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}

type FrameworkType string

const (
	FrameworkTFServing FrameworkType = "TFServing"
	FrameworkTriton    FrameworkType = "Triton"
)

type AutoScaleStrategy struct {
	MinReplicas *int32                 `json:"minReplicas,omitempty"`
	MaxReplicas *int32                 `json:"maxReplicas,omitempty"`
	AutoScaler  corev1.ObjectReference `json:"autoScaler"`
}

// +kubebuilder:object:root=true

// InferenceList contains a list of Inference
type InferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Inference `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Inference{}, &InferenceList{})
}
