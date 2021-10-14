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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ModelSpec struct {
	// Description is an arbitrary string that describes what this model for.
	Description *string `json:"description,omitempty"`
}

// ModelStatus defines the observed state of Model
type ModelStatus struct {
	// LatestVersion indicates the latest ModelVersion
	LatestVersion *VersionInfo `json:"latestVersion,omitempty"`
}

type VersionInfo struct {
	// The name of the latest ModelVersion
	ModelVersion string `json:"modelVersion,omitempty"`

	// The image name of the latest model,  e.g. "modelhub/mymodel:v9epgk"
	ImageName string `json:"imageName,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:shortName=mo
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Latest-Version",type=string,JSONPath=`.status.latestVersion.modelVersion`
// +kubebuilder:printcolumn:name="Latest-Image",type=string,JSONPath=`.status.latestVersion.imageName`

// Model is the Schema for the models API
type Model struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelSpec   `json:"spec,omitempty"`
	Status ModelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +k8s:defaulter-gen=TypeMeta
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ModelList contains a list of Model
type ModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Model `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Model{}, &ModelList{})
}
