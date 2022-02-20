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
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

// CronTemplateSpec describes a template for launching a specific job.
type CronTemplateSpec struct {
	metav1.TypeMeta `json:",inline"`

	// Workload is the specification of the desired cron job with specific types.
	// +kubebuilder:pruning:PreserveUnknownFields
	Workload *runtime.RawExtension `json:"workload,omitempty"`
}

// CronSpec defines the desired state of Cron
type CronSpec struct {

	// `CronPolicy` provides some core configurations for timing scheduling
	v1.CronPolicy `json:",inline"`

	// Specifies the job that will be created when executing a CronTask.
	CronTemplate CronTemplateSpec `json:"template"`
}

// CronStatus defines the observed state of Cron
type CronStatus struct {
	// A list of currently running jobs.
	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// History is a list of scheduled cron job with its digest records.
	// +optional
	History []CronHistory `json:"history,omitempty"`

	// Information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

type CronHistory struct {
	// Object is the reference of the historical scheduled cron job.
	Object corev1.TypedLocalObjectReference `json:"object"`
	// Status is the final status when job finished.
	Status v1.JobConditionType `json:"status"`
	// Created is the creation timestamp of job.
	Created *metav1.Time `json:"created,omitempty"`
	// Finished is the failed or succeeded timestamp of job.
	Finished *metav1.Time `json:"finished,omitempty"`
}

// +kubebuilder:subresource:status
// +kubebuilder:object:root=true

// Cron is the Schema for the crons API
type Cron struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronSpec   `json:"spec,omitempty"`
	Status CronStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CronList contains a list of Cron
type CronList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cron `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cron{}, &CronList{})
}
