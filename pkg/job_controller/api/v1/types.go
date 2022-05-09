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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobStatus represents the current observed state of the training Job.
// +k8s:deepcopy-gen=true
type JobStatus struct {
	// Conditions is an array of current observed job conditions.
	Conditions []JobCondition `json:"conditions,omitempty"`

	// ReplicaStatuses is map of ReplicaType and ReplicaStatus,
	// specifies the status of each replica.
	ReplicaStatuses map[ReplicaType]*ReplicaStatus `json:"replicaStatuses"`

	// Represents time when the job was acknowledged by the job controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the job was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Represents last time when the job was reconciled. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// ModelVersionName represents the model version name output by this job run.
	ModelVersionName string `json:"modelVersionName,omitempty"`

	// CacheBackendName is the name for the backend cache
	CacheBackendName string `json:"cacheBackendName,omitempty"`
}

// ReplicaType represents the type of the replica. Each operator needs to define its
// own set of ReplicaTypes.
type ReplicaType string

// ReplicaStatus represents the current observed state of the replica.
type ReplicaStatus struct {
	// The number of actively running pods.
	Active int32 `json:"active,omitempty"`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed,omitempty"`

	// The number of pods which reached phase Failed and reason is Evicted,
	// it is included in the number of Failed.
	Evicted int32 `json:"evicted,omitempty"`
}

// ReplicaSpec is a description of the replica.
// +k8s:deepcopy-gen=true
type ReplicaSpec struct {
	// Replicas is the desired number of replicas of the given template.
	// If unspecified, defaults to 1.
	Replicas *int32 `json:"replicas,omitempty"`

	// Spot replicas are a subset of Replicas that allow for interruptions. They can use resources with less SLO guarantee.
	// Spot replicas are suitable for stateless or fault-tolerant jobs.
	// These replicas can be scheduled with higher chance at lower resource pricing.
	SpotReplicaSpec *SpotReplicaSpec `json:"spotReplicaSpec,omitempty"`

	// Template is the object that describes the pod that
	// will be created for this replica. RestartPolicy in PodTemplateSpec
	// will be overide by RestartPolicy in ReplicaSpec
	Template v1.PodTemplateSpec `json:"template,omitempty"`

	// Restart policy for all replicas within the job.
	// One of Always, OnFailure, Never and ExitCode.
	// Default to Never.
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty"`

	// DependOn represents a list of upstream vertex conditions to be dependent on for this RepicaType to start.
	// For example, in TensorFlow workers depend on ps to start first. If not set, KubeDL will populates the
	// default DependOn based on each framework's requirements. This feature is enabled by default, and can be
	// disabled with DAGScheduling feature gate.
	DependOn []DAGCondition `json:"-"`
}

// SpotReplicaSpec is the differential spec of spot replicas, to override the replica template.
type SpotReplicaSpec struct {
	// SpotReplicaNumber is the desired number of spot replicas.
	// If unspecified, SpotReplicaNumber defaults to 0.
	// By default, replicas with index in the range from (Replicas - SpotReplicaNumber) to (Replicas -1 ) are spot replicas.
	SpotReplicaNumber int32 `json:"spotReplicaNumber,omitempty"`
	// PriorityClassName is the priorityClassName of spot replicas, to override that in replica template.
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// Labels are the extra set of labels to add on the spot replicas, overriding coinciding labels in the replica template if any.
	Labels map[string]string `json:"labels,omitempty"`
}

// JobCondition describes the state of the job at a certain point.
// +k8s:deepcopy-gen=true
type JobCondition struct {
	// Type of job condition.
	Type JobConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// JobConditionType defines all kinds of types of JobStatus.
type JobConditionType string

const (
	// JobCreated means the job has been accepted by the system,
	// but one or more of the pods/services has not been started.
	// This includes time before pods being scheduled and launched.
	JobCreated JobConditionType = "Created"

	// JobQueuing means the job has been acknowledged by controller and
	// queueing in its tenant queue, waiting to be dequeued and start to
	// reconcile.
	// The training is waiting to be scheduled by kubedl.
	JobQueuing JobConditionType = "Queuing"

	// JobRunning means all sub-resources (e.g. services/pods) of this job
	// have been successfully scheduled and launched.
	// The training is running without error.
	JobRunning JobConditionType = "Running"

	// JobRestarting means one or more sub-resources (e.g. services/pods) of this job
	// reached phase failed but maybe restarted according to it's restart policy
	// which specified by user in v1.PodTemplateSpec.
	// The training is freezing/pending.
	JobRestarting JobConditionType = "Restarting"

	// JobSucceeded means all sub-resources (e.g. services/pods) of this job
	// reached phase have terminated in success.
	// The training is complete without error.
	JobSucceeded JobConditionType = "Succeeded"

	// JobFailed means one or more sub-resources (e.g. services/pods) of this job
	// reached phase failed with no restarting.
	// The training has failed its execution.
	JobFailed JobConditionType = "Failed"
)

// SuccessPolicy is the policy to mark the job as succeeded, when the job does not contain the chief or master role.
type SuccessPolicy string

const (
	// SuccessPolicyDefault indicates the job is succeeded if all workers are succeeded or worker 0 completed
	SuccessPolicyDefault SuccessPolicy = ""
	// SuccessPolicyAllWorkers indicates the job is succeeded if all workers are succeeded.
	SuccessPolicyAllWorkers SuccessPolicy = "AllWorkers"
)

// CleanPodPolicy describes how to deal with pods when the job is finished.
type CleanPodPolicy string

const (
	CleanPodPolicyUndefined CleanPodPolicy = ""
	CleanPodPolicyAll       CleanPodPolicy = "All"
	CleanPodPolicyRunning   CleanPodPolicy = "Running"
	CleanPodPolicyNone      CleanPodPolicy = "None"
)

// RestartPolicy describes how the replicas should be restarted.
// Only one of the following restart policies may be specified.
// If none of the following policies is specified, the default one
// is RestartPolicyAlways.
type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "Always"
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	RestartPolicyNever     RestartPolicy = "Never"

	// RestartPolicyExitCode policy means that user should add exit code by themselves,
	// The job operator will check these exit codes to
	// determine the behavior when an error occurs:
	// - 1-127: permanent error, do not restart.
	// - 128-255: retryable error, will restart the pod.
	RestartPolicyExitCode RestartPolicy = "ExitCode"
)

// RunPolicy encapsulates various runtime policies of the distributed training
// job, for example how to clean up resources and how long the job can stay
// active.
// +k8s:deepcopy-gen=true
type RunPolicy struct {
	// CleanPodPolicy defines the policy to kill pods after the job completes.
	// Default to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty"`

	// TTLSecondsAfterFinished is the TTL to clean up jobs.
	// It may take extra ReconcilePeriod seconds for the cleanup, since
	// reconcile gets called periodically.
	// Default to infinite.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// Specifies the duration in seconds relative to the startTime that the job may be active
	// before the system tries to terminate it; value must be positive integer.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// Optional number of retries before marking this job failed.
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// SchedulingPolicy defines the policy related to scheduling, e.g. gang-scheduling
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"`

	// +optional
	CronPolicy *CronPolicy `json:"cronPolicy,omitempty"`
}

type CronPolicy struct {
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	// Specifies how to treat concurrent executions of a Task.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// Deadline is the timestamp that a cron job can keep scheduling util then.
	Deadline *metav1.Time `json:"deadline,omitempty"`

	// The number of finished job history to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	HistoryLimit *int32 `json:"historyLimit,omitempty"`
}

// ConcurrencyPolicy describes how the job will be handled.
// Only one of the following concurrent policies may be specified.
// If none of the following policies is specified, the default one
// is AllowConcurrent.
type ConcurrencyPolicy string

const (
	// AllowConcurrent allows CronJobs to run concurrently.
	AllowConcurrent ConcurrencyPolicy = "Allow"

	// ForbidConcurrent forbids concurrent runs, skipping next run if previous
	// hasn't finished yet.
	ForbidConcurrent ConcurrencyPolicy = "Forbid"

	// ReplaceConcurrent cancels currently running job and replaces it with a new one.
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

// SchedulingPolicy encapsulates various scheduling policies of the distributed training
// job, for example `minAvailable` for gang-scheduling.
type SchedulingPolicy struct {
	// MinAvailable is the minimum number of scheduable instances for a gang-scheduling to go.
	MinAvailable *int32 `json:"minAvailable,omitempty"`
	// The priority value. KubeDL use this field to find the priority of the job.
	// The controller populates this field from PriorityClassName by default.
	// The higher the value, the higher the priority.
	// +optional
	Priority *int32 `json:"priority,omitempty"`
	// If specified, indicates the jobs's priority. Any other name must be defined
	// by creating a PriorityClass object with that name.
	// If not specified, the job priority will be default or zero if there is no
	// default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// Queue indicates the name of tenant queue the job should be enqueued to, this field
	// can be specified manually by job owner, or partitioned by built-in plugin on basis
	// of its namespace/quota or other granularity.
	// +optional
	Queue string `json:"queue,omitempty"`
}

type DAGCondition struct {
	// Upstream defines which replica type is the source tigger.
	Upstream ReplicaType `json:"upstream"`
	// OnPhase defines at which phase the upstream replica will trigger this condition.
	OnPhase v1.PodPhase `json:"onPhase"`
}
