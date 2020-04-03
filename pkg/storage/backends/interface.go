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

package backends

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
)

// ObjectStorageBackend provides a collection of abstract methods to
// interact with different storage backends, write/read pod and job objects.
type ObjectStorageBackend interface {
	// Initialize initializes a backend storage service with local or remote
	// database.
	Initialize() error
	// Close shutdown backend storage service.
	Close() error
	// Name returns backend name.
	Name() string
	// SavePod append or update a pod record to backend, region is optional.
	SavePod(pod *v1.Pod, defaultContainerName, region string) error
	// ListPods lists pods controlled by some job, region is optional.
	ListPods(jobID, region string) ([]*dmo.Pod, error)
	// StopPod updates status of pod record as stopped.
	StopPod(ns, name, podID string) error
	// SaveJob append or update a job record to backend, region is optional.
	SaveJob(job metav1.Object, kind string, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, jobStatus *apiv1.JobStatus, region string) error
	// Get Job retrieve a job from backend, region is optional.
	GetJob(ns, name, jobID, region string) (*dmo.Job, error)
	// ListJobs lists those jobs who satisfied with query conditions.
	ListJobs(query Query) ([]*dmo.Job, error)
	// StopJob updates status of job record as stooped.
	StopJob(ns, name, jobID, region string) error
	// DeleteJob updates job as deleted from api-server, but not delete job record
	// from backend, region is optional.
	DeleteJob(ns, name, jobID, region string) error
}

// EventStorageBackend provides a collection of abstract methods to
// interact with different storage backends, write/read events.
type EventStorageBackend interface {
	// Initialize initializes a backend storage service with local or remote
	// event hub.
	Initialize() error
	// Close shutdown backend event storage service or disconnect the event hub.
	Close() error
	// Name returns backend name.
	Name() string
	// SaveEvent append or update a event record to backend.
	SaveEvent(event *v1.Event, region string) error
	// ListEvent list all events created by this job.
	ListEvent(jobNamespace, jobName string, from, to time.Time) ([]*dmo.Event, error)
}
