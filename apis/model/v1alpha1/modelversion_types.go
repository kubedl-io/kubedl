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

// ModelVersionSpec defines the desired state of ModelVersion
// ModelVersion defines the parent model, who creates the version, and an optional storage if it is different from the
// parent model.
// Each time a new ModelVersion crd is created, the ModelVersionController will create an image that incorporates the model.
type ModelVersionSpec struct {
	// The parent model name for the version
	// +required
	ModelName string `json:"modelName,omitempty"`

	// CreatedBy indicates who creates the model, e.g. If it's the tfjob that generates the model, then the CreatedBy
	// is the name of the tfjob.
	// +optional
	CreatedBy string `json:"createdBy,omitempty"`

	// Storage is the location where this version of the model is stored.
	// This is set when the storage location for this model version is different from what its parent model defined.
	// +optional
	Storage *Storage `json:"storage,omitempty"`
}

// ModelVersionStatus defines the observed state of ModelVersion
type ModelVersionStatus struct {
	// The image name of the version
	Image string `json:"image,omitempty"`

	// ImageBuildPhase is the phase of the image building process
	ImageBuildPhase ImageBuildPhase `json:"imageBuildPhase,omitempty"`

	// FinishTime is the time when image building is finished (succeeded or failed)
	FinishTime *metav1.Time `json:"finishTime,omitempty"`

	// Any message associated with the building process
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:shortName=mv
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelName`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.status.image`
// +kubebuilder:printcolumn:name="CreatedBy",type=string,JSONPath=`.spec.createdBy`
// +kubebuilder:printcolumn:name="Finish-Time",type=string,JSONPath=`.status.finishTime`

// ModelVersion is the Schema for the modelversions API
type ModelVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelVersionSpec   `json:"spec,omitempty"`
	Status ModelVersionStatus `json:"status,omitempty"`
}
type ImageBuildPhase string

const (
	ImageBuilding       ImageBuildPhase = "ImageBuilding"
	ImageBuildFailed    ImageBuildPhase = "ImageBuildFailed"
	ImageBuildSucceeded ImageBuildPhase = "ImageBuildSucceeded"
)

// +kubebuilder:object:root=true
// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:resource:scope=Namespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ModelVersionList contains a list of ModelVersion
type ModelVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ModelVersion{}, &ModelVersionList{})
}
