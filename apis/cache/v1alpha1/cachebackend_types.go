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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KUBEDL_CACHE_NAME = "KUBEDL_CACHE_NAME"
)

// CacheBackendSpec defines the desired state of CacheBackend
type CacheBackendSpec struct {
	// CacheBackendName is equal to ObjectMeta.Name for now
	CacheBackendName string `json:"name,omitempty"`

	// The location in the container where the dataset should be mounted
	MountPath string `json:"mountPath,omitempty"`

	// Dataset describes the parameters related to the dataset file
	Dataset *Dataset `json:"dataset,omitempty"`

	// CacheEngine is different kinds of cache engine
	CacheEngine *CacheEngine `json:"cacheEngine,omitempty"`

	// Options is used to set additional configuration
	Options Options `json:"options,omitempty"`
}

// CacheBackendStatus defines the observed state of CacheBackend
type CacheBackendStatus struct {
	// CacheEngine is the current cache engine used by CacheBackend
	CacheEngine string `json:"cacheEngine,omitempty"`

	// CacheStatus is used to help understand the status of a caching process
	CacheStatus CacheStatus `json:"cacheStatus,omitempty"`

	// UsedBy array contains the jobs currently using this cacheBackends
	UsedBy []string `json:"usedBy,omitempty"`

	// UsedNum equals to the size of UsedBy array
	UsedNum int `json:"usedNum,omitempty"`

	// LastUsedTime is equal to the completion time of the last job that used CacheBackend
	LastUsedTime *metav1.Time `json:"lastUsedTime,omitempty"`
}

// Dataset is used to define where specific data sources are stored and mounted
// A job may have multiple mount points, so it is a list
type Dataset struct {
	// DataSources is a list of dataset path
	DataSources []DataSource `json:"dataSources,omitempty"`
}

type DataSource struct {
	// Location is the dataset path in variety file system
	Location string `json:"location,omitempty"`

	// SubdirectoryName is used as the file name of the data source in mountPath.
	// The directory structure in container is as follows:
	// - MountPath
	//   - SubDirName1
	//   - SubDirName2
	//   - ...
	SubDirName string `json:"subDirName,omitempty"`
}

type CacheEngine struct {
	// Fluid may be the only caching engine for the time being
	Fluid *Fluid `json:"fluid,omitempty"`
}

type Fluid struct {
	// AlluxioRuntime is used to configure the cache runtime
	// If this parameter is not specified, the cache will not take effect and the data set will be directly mounted
	AlluxioRuntime *AlluxioRuntime `json:"alluxioRuntime,omitempty"`
}

type AlluxioRuntime struct {
	// Replicas is the min replicas of dataset in the cluster
	Replicas int32 `json:"replicas,omitempty"`

	// Fluid supports multi-tier configuration
	TieredStorage []Level `json:"tieredStorage,omitempty"`
}

type Level struct {
	// For Alluxio, it will use the cache path directory as its cache store
	CachePath string `json:"cachePath,omitempty"`

	// Quota defines a cache capacity of the directory
	Quota string `json:"quota,omitempty"`

	// Fluid sorts levels according to mediumType
	MediumType string `json:"mediumType,omitempty"`
}

type CacheStatus string

// The four CacheStatus vary as follows:
// CacheCreating -> PVCCreating ( CacheSucceed ) -> PVCCreated
//
//	or -> CacheFailed
const (
	PVCCreated    CacheStatus = "PVCCreated"
	PVCCreating   CacheStatus = "PVCCreating"
	CacheFailed   CacheStatus = "CacheFailed"
	CacheCreating CacheStatus = "CacheCreating"
)

type Options struct {
	// IdleTime means how long cacheBackend is not currently in use
	IdleTime time.Duration `json:"idleTime,omitempty"`
}

// +k8s:defaulter-gen=TypeMeta
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.status.cacheEngine`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.cacheStatus`
// +kubebuilder:printcolumn:name="Used-Num",type=integer,JSONPath=`.status.usedNum`
// +kubebuilder:printcolumn:name="Last-Used-Time",type=date,JSONPath=`.status.lastUsedTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// CacheBackend is the Schema for the cache backends API
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
