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
	MountCachePVC = "MountCachePVC"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CacheBackendSpec defines the desired state of CacheBackend
type CacheBackendSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The location in the container where the dataset should be mounted
	MountPath string `json:"mountPath,omitempty"`

	// Provide multiple Caching mode to user
	CacheMode *CacheMode `json:"cacheMode,omitempty"`

	// CacheEngine is different kinds of cache engine
	CacheEngine *CacheEngine `json:"cacheEngine,omitempty"`
}

// CacheBackendStatus defines the observed state of CacheBackend
type CacheBackendStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PVCName is the PVC being created name
	// the string will be associated with the job which create cache
	PVCName string `json:"pvcName,omitempty"`

	// CacheStatus displays the status of the entire caching process
	CacheStatus CacheStatus `json:"cacheStatus,omitempty"`

	MountStatus MountStatus `json:"mountStatus,omitempty"`

	// FinishTime is the time when data cache is finished.
	FinishTime *metav1.Time `json:"finishTime,omitempty"`

	// Message is some information in the cache process, such as whether the PVC was created or not
	Message string `json:"message,omitempty"`
}

type CacheMode string

const (
	BestEffort CacheMode = "BestEffort"
)

type CacheEngine struct {
	// Fluid may be the only caching engine for the time being
	Fluid *Fluid `json:"fluid,omitempty"`
}

type Fluid struct {
	// Dataset describes the parameters related to the dataset file
	// +required
	Dataset *Dataset `json:"dataset,omitempty"`

	// AlluxioRuntime is used to configure the cache runtime
	// If this parameter is not specified, the cache will not take effect and the data set will be directly mounted
	// +optional
	AlluxioRuntime *AlluxioRuntime `json:"alluxioRuntime,omitempty"`
}

// Dataset is used to define where specific data sources are stored and mounted
// A job may have multiple mount points, so it is a list
type Dataset struct {
	// Mounts is a list of mount points
	Mounts []MountPoint `json:"mounts,omitempty"`
}

type MountPoint struct {
	// The dataset path in variety file system
	DataSource string `json:"dataSource,omitempty"`
}

type AlluxioRuntime struct {
	// Fluid supports multi-tier configuration
	TierdStorage []Level `json:"tierdStorage,omitempty"`
}

type Level struct {
	// Alluxio will use the cache path directory as its cache store
	CachePath string `json:"cachePath,omitempty"`

	// Quota defines a cache capacity of the directory
	Quota string `json:"quota,omitempty"`

	// Fluid sorts levels according to mediumType
	MediumType string `json:"mediumType,omitempty"`
}

type CacheStatus string

const (
	CacheCreated   CacheStatus = "CacheStatus"
	Caching        CacheStatus = "Caching"
	CacheFailed    CacheStatus = "CacheFailed"
	CacheSucceeded CacheStatus = "CacheSucceeded"
)

type MountStatus string

const (
	WaitPVCCreated MountStatus = "WaitPVCCreated"
	Mounting       MountStatus = "Mounting"
	MountFailed    MountStatus = "MountFailed"
	MountSucceeded MountStatus = "MountSucceeded"
)

//+k8s:defaulter-gen=TypeMeta
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CacheBackend is the Schema for the cachebackends API
type CacheBackend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CacheBackendSpec   `json:"spec,omitempty"`
	Status CacheBackendStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CacheBackendList contains a list of CacheBackend
type CacheBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CacheBackend `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CacheBackend{}, &CacheBackendList{})
}
