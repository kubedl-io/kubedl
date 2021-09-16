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

const (
	// KubeDLModelPath is the env key to indicate model path in container.
	KubeDLModelPath = "KUBEDL_MODEL_PATH"

	// DefaultModelPathInImage is the default path where model artifacts are stored inside the model image generated by Kaniko.
	DefaultModelPathInImage = "/kubedl-model"
)

// ModelVersionSpec defines a particular version of the Model.
// Each time a new ModelVersion crd is created, the ModelVersionController will create an image that incorporates the model.
type ModelVersionSpec struct {
	// The Model name for this ModelVersion
	// +required
	ModelName string `json:"modelName,omitempty"`

	// CreatedBy indicates the entity that creates the model version. e.g. the name of the tfjob that creates the ModelVersion. the name
	// of the user that creates the ModelVersion
	// +optional
	CreatedBy string `json:"createdBy,omitempty"`

	// Storage is the location where this ModelVersion is stored.
	// +optional
	Storage *Storage `json:"storage,omitempty"`

	// ImageRepo is the image repository to push the generated model image. e.g. docker hub.  "modelhub/mymodel"
	// A particular image tag is automatically generated by the system. e.g.  "modelhub/mymodel:v9epgk"
	// +required
	ImageRepo string `json:"imageRepo,omitempty"`
}

// ModelVersionStatus defines the observed state of ModelVersion
type ModelVersionStatus struct {
	// The image name of the model version
	Image string `json:"image,omitempty"`

	// ImageBuildPhase is the phase of the image building process
	ImageBuildPhase ImageBuildPhase `json:"imageBuildPhase,omitempty"`

	// FinishTime is the time when image building is finished.
	FinishTime *metav1.Time `json:"finishTime,omitempty"`

	// Any message associated with the building process
	Message string `json:"message,omitempty"`
}

type Storage struct {
	// NFS represents the alibaba cloud nas storage
	NFS *NFS `json:"nfs,omitempty"`

	// LocalStorage represents the local host storage
	LocalStorage *LocalStorage `json:"localStorage,omitempty"`

	// AWSEfs represents the AWS Elastic FileSystem
	AWSEfs *AWSEfs `json:"AWSEfs,omitempty"`
}

// LocalStorage defines the local storage for storing the model version.
// For a distributed training job, the nodeName will be the node where the chief/master worker run to export the model.
type LocalStorage struct {
	// The local host path to export the model.
	// +required
	Path string `json:"path,omitempty"`

	// The mounted path inside the container.
	// The training code is expected to export the model artifacts under this path, such as storing the tensorflow saved_model.
	MountPath string `json:"mountPath,omitempty"`

	// The name of the node for storing the model. This node will be where the chief worker run to export the model.
	// +required
	NodeName string `json:"nodeName,omitempty"`
}

type AWSEfs struct {
	// VolumeHandle indicates the backend EFS volume. Check the link for details
	// https://github.com/kubernetes-sigs/aws-efs-csi-driver/tree/master/examples/kubernetes
	// It is of the form "[FileSystemId]:[Subpath]:[AccessPointId]"
	// e.g. FilesystemId with subpath and access point Id:  fs-e8a95a42:/my/subpath:fsap-19f752f0068c22464.
	// FilesystemId with access point Id:   fs-e8a95a42::fsap-068c22f0246419f75
	// FileSystemId with subpath: 	 fs-e8a95a42:/dir1
	VolumeHandle string `json:"volumeHandle,omitempty"`

	// The attributes passed to the backend EFS
	Attributes map[string]string `json:"attributes,omitempty"`
}

// NFS represents the Alibaba Cloud Nas storage
type NFS struct {
	// NFS server address, e.g. "***.cn-beijing.nas.aliyuncs.com"
	Server string `json:"server,omitempty"`

	// The path under which the model is stored, e.g. /models/my_model1
	Path string `json:"path,omitempty"`

	// The mounted path inside the container.
	// The training code is expected to export the model artifacts under this path, such as storing the tensorflow saved_model.
	MountPath string `json:"mountPath,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:shortName=mv
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelName`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.status.image`
// +kubebuilder:printcolumn:name="Created-By",type=string,JSONPath=`.spec.createdBy`
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
