/*
Copyright 2021 The KubeDL Authors.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NotebookNameLabel     = "notebook.kubedl.io/notebook-name"
	NotebookContainerName = "notebook"
	NotebookPortName      = "nb-port"
	NotebookDefaultPort   = 8888
	NotebookKind          = "Notebook"
)

// NotebookSpec defines the desired state of Notebook
type NotebookSpec struct {

	// Template describes the pod template for notebook
	Template v1.PodTemplateSpec `json:"template,omitempty"`
}

// NotebookStatus defines the observed state of Notebook
type NotebookStatus struct {
	// Notebook condition
	Condition NotebookCondition `json:"condition,omitempty"`

	// Message associated with the condition
	Message string `json:"message,omitempty"`

	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// URL for the notebook
	Url string `json:"url,omitempty"`
}

type NotebookCondition string

const (
	NotebookCreated NotebookCondition = "Created"

	NotebookRunning NotebookCondition = "Running"

	NotebookTerminated NotebookCondition = "Terminated"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:shortName=nb
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.condition`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`

// Notebook is the Schema for the notebooks API
type Notebook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotebookSpec   `json:"spec,omitempty"`
	Status NotebookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +k8s:defaulter-gen=TypeMeta
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotebookList contains a list of Notebook
type NotebookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notebook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notebook{}, &NotebookList{})
}
