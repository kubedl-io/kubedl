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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ModelSpec defines the desired state of Model
// Model defines the storage where the model is located, the imageRepo for pushing the generated model container.
// Each model has a series of ModelVersion associated with it.
type ModelSpec struct {
	// Storage is the location where the model is stored.
	// +optional
	Storage *Storage `json:"storage,omitempty"`

	// ImageRepo is the image repository to push the generated image. e.g. docker hub.  "kubernetes/pause"
	// +required
	ImageRepo string `json:"imageRepo,omitempty"`
}

type Storage struct {
	AliCloudNas  *AliCloudNas  `json:"aliCloudNas,omitempty"`
	LocalStorage *LocalStorage `json:"localStorage,omitempty"`
}

// If the localStorage is defined in parent model, nodeName must be set, that is, all output models will be stored on the same node.
// And all subsequent modelVersion will inherit the localStorage from parent model.
// If the localStorage is defined in modelVersion, it will overwrite the parent model LocalStorage if any.
type LocalStorage struct {
	// The local path on the host
	// +required
	Path string `json:"path,omitempty"`

	// The node for storing the model. This node will be where the chief worker run to output the model.
	// +required
	NodeName string `json:"nodeName,omitempty"`
}

type AliCloudNas struct {
	// Nas server address
	Server string `json:"server,omitempty"`

	// The path under which the model is stored, e.g. /models/my_model1
	Path       string            `json:"path,omitempty"`
	Vers       string            `json:"vers,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ModelStatus defines the observed state of Model
type ModelStatus struct {
	LatestVersion *VersionInfo `json:"latestVersion,omitempty"`
}

type VersionInfo struct {
	// The name of the latest ModelVersion
	ModelVersion string `json:"modelVersion,omitempty"`

	// The image name of the latest model
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
