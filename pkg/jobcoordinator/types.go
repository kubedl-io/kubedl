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

package jobcoordinator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
)

type Plugin interface {
	Name() string
}

type PreFilterPlugin interface {
	PreFilter(ctx context.Context, qu *QueueUnit) Status
	Plugin
}

type FilterPlugin interface {
	Filter(ctx context.Context, qu *QueueUnit) Status
	Plugin
}

type ScorePlugin interface {
	Score(ctx context.Context, qu *QueueUnit) (int64, Status)
	Plugin
}

type PreDequeuePlugin interface {
	PreDequeue(ctx context.Context, qu *QueueUnit) Status
	Plugin
}

type TenantPlugin interface {
	TenantName(qu *QueueUnit) string
	Plugin
}

type PluginFactory func(client client.Client, recorder record.EventRecorder) Plugin

type Status interface {
	// Code returns code of the status.
	Code() Code
	// Message returns a concatenated message on reasons of the status.
	Message() string
	// Reasons returns reasons of the status.
	Reasons() []string
	// AppendReason appends given reason to the status.
	AppendReason(reason string)
	// IsSuccess returns true if and only if "status" is nil or Code is "Success".
	IsSuccess() bool
	// IsUnschedulable returns true if "status" is Unschedulable (Unschedulable or UnschedulableAndUnresolvable).
	IsUnschedulable() bool
	// AsError returns nil if the status is a success; otherwise returns an "error" object
	// with a concatenated message on reasons of the status.
	AsError() error
}

type CoordinatorConfiguration struct {
	SchedulePeriod    time.Duration
	TenantPlugin      string
	PreFilterPlugins  []string
	FilterPlugins     []string
	ScorePlugins      []string
	PreDequeuePlugins []string
}

type QueueUnit struct {
	Tenant        string
	Priority      *int32
	Object        client.Object
	SchedPolicy   *v1.SchedulingPolicy
	Specs         map[v1.ReplicaType]*v1.ReplicaSpec
	Status        *v1.JobStatus
	Resources     corev1.ResourceList
	SpotResources corev1.ResourceList
	Owner         workqueue.RateLimitingInterface
}

func (qu *QueueUnit) Key() string {
	return fmt.Sprintf("%s/%s/%s",
		qu.Object.GetObjectKind().GroupVersionKind().Kind, qu.Object.GetNamespace(), qu.Object.GetName())
}

// SpotQueueUnit returns a wrapped queue unit which takes spot resource as
// its total resource requests and shallow copy other fields as its properties.
func (qu *QueueUnit) SpotQueueUnit() *QueueUnit {
	return &QueueUnit{
		Tenant:      qu.Tenant,
		Priority:    qu.Priority,
		Object:      qu.Object,
		SchedPolicy: qu.SchedPolicy,
		Specs:       qu.Specs,
		Status:      qu.Status,
		Resources:   qu.SpotResources,
		Owner:       qu.Owner,
	}
}

type QueueUnitScore struct {
	QueueUnit *QueueUnit
	Score     int64
}

// status indicates the result of running a plugin. It consists of a code, a
// message and (optionally) an error. When the status code is not `Success`,
// the reasons should explain why.
// NOTE: A nil status is also considered as Success.
type status struct {
	code    Code
	reasons []string
	err     error
}

func (s *status) Code() Code {
	if s == nil {
		return Success
	}
	return s.code
}

func (s *status) Message() string {
	if s == nil {
		return ""
	}
	return strings.Join(s.reasons, ", ")
}

func (s *status) Reasons() []string {
	return s.reasons
}

func (s *status) AppendReason(reason string) {
	s.reasons = append(s.reasons, reason)
}

func (s *status) IsSuccess() bool {
	return s.Code() == Success
}

func (s *status) IsUnschedulable() bool {
	code := s.Code()
	return code == Unschedulable
}

func (s *status) AsError() error {
	if s.IsSuccess() {
		return nil
	}
	if s.err != nil {
		return s.err
	}
	return errors.New(s.Message())
}

// NewStatus makes a status out of the given arguments and returns its pointer.
func NewStatus(code Code, reasons ...string) Status {
	s := &status{
		code:    code,
		reasons: reasons,
	}
	if code == Error {
		s.err = errors.New(s.Message())
	}
	return s
}

// Code is the status code/type which is returned from plugins.
type Code int

// These are predefined codes used in a status.
const (
	// Success means that plugin ran correctly and found job schedulable.
	// NOTE: A nil status is also considered as "Success".
	Success Code = iota
	// Error is used for internal plugin errors, unexpected input, etc.
	Error
	// Unschedulable is used when a plugin finds a job unschedulable.
	// The accompanying status message should explain why the pod is unschedulable.
	Unschedulable
	// Wait is used when a Permit plugin finds a pod scheduling should wait.
	Wait
	// Skip is used when a Bind plugin chooses to skip binding.
	Skip
)
