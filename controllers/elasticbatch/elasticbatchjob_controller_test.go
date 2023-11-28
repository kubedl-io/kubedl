// Copyright 2022 The Alibaba Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package elasticbatch

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/workqueue"

	"github.com/alibaba/kubedl/apis"
	inference "github.com/alibaba/kubedl/apis/inference/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
	"github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	// Enable klog which is used in dependencies
	_ = pflag.Set("logtostderr", "true")
	_ = pflag.Set("v", "10")
}

type ElasticBatchJobReconcilerTest struct {
	ElasticBatchJobReconciler
}

func (r *ElasticBatchJobReconcilerTest) GetJobFromAPIClient(namespace, name string) (client.Object, error) {
	job := &inference.ElasticBatchJob{}
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	return job, err
}

type FakeJobExpectations struct {
	*k8scontroller.ControllerExpectations
}

func (fe FakeJobExpectations) SatisfiedExpectations(controllerKey string) bool {
	// alwasys return true, so that, reconcile loop can always trigger sync,
	return true
}

func tearDown() {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconcilerTest(client client.Client, scheme *runtime.Scheme,
	recorder record.EventRecorder,
	config options.JobControllerConfiguration) *ElasticBatchJobReconcilerTest {
	r := &ElasticBatchJobReconcilerTest{
		ElasticBatchJobReconciler{
			Client: client,
			scheme: scheme,
		},
	}
	r.recorder = recorder
	// Initialize pkg job controller with components we only need.
	r.ctrl = job_controller.JobController{
		Client:             client,
		APIReader:          client,
		BackoffStatesQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Controller:         r,
		PodControl:         job_controller.NewPodControl(client, recorder),
		ServiceControl:     job_controller.NewServiceControl(client, recorder),
		Config:             config,
		Recorder:           recorder,
		Metrics:            metrics.NewJobMetrics(inference.ElasticBatchJobKind, client),
	}
	if r.ctrl.Config.EnableGangScheduling {
		r.ctrl.GangScheduler = registry.Get(r.ctrl.Config.GangSchedulerName)
	}
	r.ctrl.Expectations = FakeJobExpectations{ControllerExpectations: k8scontroller.NewControllerExpectations()}
	return r
}

// Test Scenario: check the job is succeeded only if all workers are succeeded
// 1. Create a job with 2 replicas
// 2. Mark the 2 pods as running, and check the job is running
// 3. Mark worker0 as succeeded, the job should still be running, because the successPolicy is AllWorkers
// 4. Mark worker1 as succeeded, now assert the job should be succeeded
func TestAllWorkersSuccessPolicy(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = apis.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	defer tearDown()

	// a job with 2 replicas
	elasticbatchJob := createElasticBatchJob("job1", 2)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(elasticbatchJob).Build()
	jobControllerConfig := options.JobControllerConfiguration{}
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme, corev1.EventSource{Component: "broadcast-controller"})
	elasticbatchJobReconciler := NewReconcilerTest(fakeClient, scheme, recorder, jobControllerConfig)

	jobRequest := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "job1",
			Namespace: "default",
		},
	}
	// reconcile the job, it should create 2 replicas
	_, _ = elasticbatchJobReconciler.Reconcile(context.Background(), jobRequest)

	markPodStatus("job1-aimaster-0", corev1.PodRunning, elasticbatchJobReconciler)
	// mark two pods running
	markPodStatus("job1-worker-0", corev1.PodRunning, elasticbatchJobReconciler)
	markPodStatus("job1-worker-1", corev1.PodRunning, elasticbatchJobReconciler)

	// Reconcile again, the job should go into Running state
	_, _ = elasticbatchJobReconciler.Reconcile(context.Background(), jobRequest)
	_ = elasticbatchJobReconciler.Get(context.TODO(), jobRequest.NamespacedName, elasticbatchJob)
	assert.True(t, util.HasCondition(elasticbatchJob.Status, v1.JobRunning))

	// make job1-worker-0 succeed
	markPodStatus("job1-worker-0", corev1.PodSucceeded, elasticbatchJobReconciler)

	// reconcile again
	_, _ = elasticbatchJobReconciler.Reconcile(context.Background(), jobRequest)
	// one worker succeeded, because of AllWorker SuccessPolicy, the job is still running
	_ = elasticbatchJobReconciler.Get(context.TODO(), jobRequest.NamespacedName, elasticbatchJob)
	assert.True(t, util.HasCondition(elasticbatchJob.Status, v1.JobRunning))

	// mark job1-worker-0 succeed too
	markPodStatus("job1-worker-1", corev1.PodSucceeded, elasticbatchJobReconciler)

	// reconcile again
	_, _ = elasticbatchJobReconciler.Reconcile(context.Background(), jobRequest)

	// two workers succeeded, the jobs is succeeded
	_ = elasticbatchJobReconciler.Get(context.TODO(), jobRequest.NamespacedName, elasticbatchJob)
	assert.True(t, util.HasCondition(elasticbatchJob.Status, v1.JobSucceeded))
}

func markPodStatus(podName string, status corev1.PodPhase, elasticbatchJobReconciler *ElasticBatchJobReconcilerTest) {
	worker := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      podName,
			Namespace: "default",
		},
	}
	pod := &corev1.Pod{}
	_ = elasticbatchJobReconciler.Get(context.TODO(), worker.NamespacedName, pod)

	var containerState corev1.ContainerState
	switch status {
	case corev1.PodSucceeded:
		containerState = corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
			},
		}
	case corev1.PodRunning:
		containerState = corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{
				StartedAt: metav1.Now(),
			},
		}
	}

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			Name:  "elasticbatch",
			State: containerState,
		},
	}
	pod.Status.Phase = status
	if status == corev1.PodRunning {
		now := metav1.Now()
		pod.Status.StartTime = &now
	}
	_ = elasticbatchJobReconciler.Status().Update(context.Background(), pod)
}

func createElasticBatchJob(jobName string, replicas int32) *inference.ElasticBatchJob {
	var aimasterReplica int32 = 1

	successPolicy := v1.SuccessPolicyAllWorkers
	elasticbatchJob1 := &inference.ElasticBatchJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   "default",
			UID:         "12345",
			Annotations: map[string]string{"aimaster": "ready"},
		},

		Spec: inference.ElasticBatchJobSpec{
			ElasticBatchReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
				"AIMaster": {
					Replicas:      &aimasterReplica,
					RestartPolicy: "Never",
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: jobName + "-aimaster-pod",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "elasticbatch",
									Image: "kubedl/aimaster:latest",
								},
							},
						},
					},
				},
				"Worker": {
					Replicas:      &replicas,
					RestartPolicy: "Never",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "elasticbatch",
									Image: "kubedl/elasticbatch:1.0",
								},
							},
						},
					},
				},
			},
			SuccessPolicy: &successPolicy,
		},
		Status: v1.JobStatus{},
	}
	return elasticbatchJob1
}
