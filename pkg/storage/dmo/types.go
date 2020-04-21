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

package dmo

import (
	"time"

	v1 "k8s.io/api/core/v1"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/jinzhu/gorm"
)

// Pod contains fields collected from original Pod object and extra info that
// we concerned about, they will be persisted by storage backend.
type Pod struct {
	// Primary ID auto incremented by underlying database.
	ID uint64 `gorm:"column:id;not null;AUTO_INCREMENT;primary_key" json:"id"`
	// Metadata we concerned aggregated from pod object.
	Name      string      `gorm:"type:varchar(128);column:name" json:"name"`
	Namespace string      `gorm:"type:varchar(128);column:namespace" json:"namespace"`
	PodID     string      `gorm:"type:varchar(64);column:pod_id" json:"pod_id"`
	Version   string      `gorm:"type:varchar(32);column:version" json:"version"`
	Status    v1.PodPhase `gorm:"type:varchar(32);column:status" json:"status"`
	Image     string      `gorm:"type:varchar(255);column:image" json:"image"`
	// Job ID of this pod controlled by.
	JobID string `gorm:"type:varchar(64);column:job_id" json:"job_id"`
	// Replica type of this pod figured in training job.
	ReplicaType string `gorm:"type:varchar(32);column:replica_type" json:"replica_type"`
	// Resources this pod requested, marshaled from a ResourceRequirements object.
	Resources string `gorm:"type:varchar(1024);column:resources" json:"resources"`
	// IP information allocated for this pod.
	HostIP *string `gorm:"type:varchar(64);column:host_ip" json:"host_ip,omitempty"`
	PodIP  *string `gorm:"type:varchar(64);column:pod_ip" json:"pod_ip,omitempty"`
	// DeployRegion indicates the physical region(IDC) this pod located in,
	DeployRegion *string `gorm:"type:varchar(64);column:deploy_region" json:"deploy_region,omitempty"`
	// Deleted indicates that whether this pod has been deleted or not.
	Deleted *int `gorm:"type:tinyint(4);column:deleted" json:"is_del,omitempty"`
	// IsInEtcd indicates that whether record of this pod has been removed from etcd.
	// Deleted pod could stay up in etcd due to different runtime policies.
	IsInEtcd *int `gorm:"type:tinyint(4);column:is_in_etcd" json:"is_in_etcd,omitempty"`
	// Optional remark text reserved.
	Remark *string `gorm:"type:text;column:remark" json:"remark,omitempty"`
	// Timestamps of different pod phases and status transitions.
	GmtCreated  time.Time  `gorm:"type:datetime;column:gmt_created" json:"gmt_created"`
	GmtModified time.Time  `gorm:"type:datetime;column:gmt_modified" json:"gmt_modified"`
	GmtStarted  *time.Time `gorm:"type:datetime;column:gmt_started" json:"gmt_started,omitempty"`
	GmtFinished *time.Time `gorm:"type:datetime;column:gmt_finished" json:"gmt_finished,omitempty"`
}

// Job contains fields collected from original Job object and extra info that
// we concerned about, they will be persisted by storage backend.
type Job struct {
	// Primary ID auto incremented by underlying database.
	ID uint64 `gorm:"column:id;not null;AUTO_INCREMENT;primary_key" json:"id"`
	// Metadata we concerned aggregated from job object.
	Name      string                 `gorm:"type:varchar(128);column:name" json:"name"`
	Namespace string                 `gorm:"type:varchar(128);column:namespace" json:"namespace"`
	JobID     string                 `gorm:"type:varchar(64);column:job_id" json:"job_id"`
	Version   string                 `gorm:"type:varchar(32);column:version" json:"version"`
	Status    apiv1.JobConditionType `gorm:"type:varchar(32);column:status" json:"status"`
	// Kind of this job: TFJob, PytorchJob...
	Kind string `gorm:"type:varchar(32);column:kind" json:"kind"`
	// Resources this job requested, including replicas and resources of each type,
	// it's formatted as follows:
	// {
	//   "PS": {
	//     "replicas": 1,
	//     "resources": {"cpu":2, "memory": "10Gi"}
	//   },
	//   "Worker": {
	//     "replicas": 2,
	//     "resources": {"cpu":2, "memory": "10Gi"}
	//   }
	// }
	Resources string `gorm:"type:text;column:resources" json:"resources"`
	// DeployRegion indicates the physical region(IDC) this job located in,
	// reserved for jobs running in across-region-clusters.
	DeployRegion *string `gorm:"type:varchar(64);column:deploy_region" json:"deploy_region,omitempty"`
	// Fields reserved for multi-tenancy job management scenarios, indicating
	// which tenant this job belongs to and who's the owner(user).
	Tenant *string `gorm:"type:varchar(255);column:tenant" json:"tenant,omitempty"`
	Owner  *string `gorm:"type:varchar(255);column:owner" json:"owner,omitempty"`
	// Deleted indicates that whether this job has been deleted or not.
	Deleted *int `gorm:"type:tinyint(4);column:deleted" json:"deleted,omitempty"`
	// IsInEtcd indicates that whether record of this job has been removed from etcd.
	// Deleted job could stay up in etcd due to different runtime policies.
	IsInEtcd *int `gorm:"type:tinyint(4);column:is_in_etcd" json:"is_in_etcd,omitempty"`
	// Timestamps of different job phases and status transitions.
	GmtCreated  time.Time  `gorm:"type:datetime;column:gmt_created" json:"gmt_created"`
	GmtModified time.Time  `gorm:"type:datetime;column:gmt_modified" json:"gmt_modified"`
	GmtFinished *time.Time `gorm:"type:datetime;column:gmt_finished" json:"gmt_finished,omitempty"`
}

// Event contains fields collected from original Event object, they will be persisted
// by storage backend.
type Event struct {
	// Name of this event.
	Name string `gorm:"type:varchar(128);column:name" json:"name"`
	// Kind of object involved by event.
	Kind string `gorm:"type:varchar(32);column:kind" json:"kind"`
	// Type of this event.
	Type string `gorm:"type:varchar(32);column:type" json:"type"`
	// Involved Object Namespace.
	ObjNamespace string `gorm:"type:varchar(64);column:obj_namespace" json:"obj_namespace"`
	// Involved Object Name.
	ObjName string `gorm:"type:varchar(64);column:obj_name" json:"obj_name"`
	// Involved Object UID.
	ObjUID string `gorm:"type:varchar(64);column:obj_uid" json:"obj_uid"`
	// Reason(short, machine understandable string) of this event.
	Reason string `gorm:"type:varchar(128);column:reason" json:"reason"`
	// Message(long, human understandable description) of this event.
	Message string `gorm:"type:text;column:message" json:"message"`
	// Number of times this event has occurred.
	Count int32 `gorm:"type:integer(32);column:reason" json:"count"`
	// Region indicates the physical region(IDC) this job located in.
	Region *string `gorm:"type:varchar(64);column:region" json:"region,omitempty"`
	// The time at which the event was first recorded.
	FirstTimestamp time.Time `gorm:"type:datetime;column:first_timestamp" json:"first_timestamp"`
	// The time at which the most recent occurrence of this event was recorded.
	LastTimestamp time.Time `gorm:"type:datetime;column:last_timestamp" json:"last_timestamp"`
}

func (pod Pod) TableName() string {
	return "replica_info"
}

// BeforeCreate update gmt_modified timestamp.
func (pod *Pod) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("gmt_modified", time.Now().UTC())
}

// BeforeUpdate update gmt_modified timestamp.
func (pod *Pod) BeforeUpdate(scope *gorm.Scope) error {
	return scope.SetColumn("gmt_modified", time.Now().UTC())
}

func (job Job) TableName() string {
	return "job_info"
}

// BeforeUpdate update gmt_modified timestamp.
func (job *Job) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("gmt_modified", time.Now())
}

// BeforeUpdate update gmt_modified timestamp.
func (job *Job) BeforeUpdate(scope *gorm.Scope) error {
	return scope.SetColumn("gmt_modified", time.Now())
}

func (e Event) TableName() string {
	return "event_info"
}
