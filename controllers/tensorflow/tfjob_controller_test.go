package tensorflow

import (
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/alibaba/kubedl/api"
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/job_controller"
	"github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	// Enable klog which is used in dependencies
	_ = flag.Set("logtostderr", "true")
	_ = flag.Set("v", "10")
}

type TFJobReconcilerTest struct {
	TFJobReconciler
}

func (r *TFJobReconcilerTest) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &tfv1.TFJob{}
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	return job, err
}

func (r *TFJobReconcilerTest) satisfiedExpectations(tfJob *tfv1.TFJob) bool {
	// during unit test, no watch events will happen, hence always return true to trigger reconcile
	return true
}

type FakeJobExpectations struct {
	*k8scontroller.ControllerExpectations
}

func (fe FakeJobExpectations) SatisfiedExpectations(controllerKey string) bool {
	// alwasys return true, so that, reconcile loop can always trigger sync,
	return true
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconcilerTest(client client.Client, scheme *runtime.Scheme,
	recorder record.EventRecorder,
	config job_controller.JobControllerConfiguration) *TFJobReconcilerTest {
	r := &TFJobReconcilerTest{
		TFJobReconciler{
			Client: client,
			scheme: scheme,
		},
	}
	r.recorder = recorder
	// Initialize pkg job controller with components we only need.
	r.ctrl = job_controller.JobController{
		Controller:         r,
		Expectations:       k8scontroller.NewControllerExpectations(),
		Config:             config,
		BackoffStatesQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Recorder:           r.recorder,
		Metrics:            metrics.NewJobMetrics(tfv1.Kind, r.Client),
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
	_ = api.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// a job with 2 replicas
	tfjob := createTFJob("job1", 2)
	fakeClient := fake.NewFakeClientWithScheme(scheme, tfjob)
	jobControllerConfig := job_controller.JobControllerConfiguration{}
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme, corev1.EventSource{Component: "broadcast-controller"})
	tfJobReconciler := NewReconcilerTest(fakeClient, scheme, recorder, jobControllerConfig)

	jobRequest := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "job1",
			Namespace: "default",
		},
	}
	// reconcile the job, it should create 2 replicas
	_, _ = tfJobReconciler.Reconcile(jobRequest)

	// mark two pods running
	markPodStatus("job1-worker-0", corev1.PodRunning, tfJobReconciler)
	markPodStatus("job1-worker-1", corev1.PodRunning, tfJobReconciler)

	// Reconcile again, the job should go into Running state
	_, _ = tfJobReconciler.Reconcile(jobRequest)
	_ = tfJobReconciler.Get(context.TODO(), jobRequest.NamespacedName, tfjob)
	assert.True(t, util.HasCondition(tfjob.Status, v1.JobRunning))

	// make job1-worker-0 succeed
	markPodStatus("job1-worker-0", corev1.PodSucceeded, tfJobReconciler)

	// reconcile again
	_, _ = tfJobReconciler.Reconcile(jobRequest)
	// one worker succeeded, because of AllWorker SuccessPolicy, the job is still running
	_ = tfJobReconciler.Get(context.TODO(), jobRequest.NamespacedName, tfjob)
	assert.True(t, util.HasCondition(tfjob.Status, v1.JobRunning))

	// mark job1-worker-0 succeed too
	markPodStatus("job1-worker-1", corev1.PodSucceeded, tfJobReconciler)

	// reconcile again
	_, _ = tfJobReconciler.Reconcile(jobRequest)

	// two workers succeeded, the jobs is succeeded
	_ = tfJobReconciler.Get(context.TODO(), jobRequest.NamespacedName, tfjob)
	assert.True(t, util.HasCondition(tfjob.Status, v1.JobSucceeded))
}

func markPodStatus(podName string, status corev1.PodPhase, tfJobReconciler *TFJobReconcilerTest) {
	worker := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      podName,
			Namespace: "default",
		},
	}
	pod := &corev1.Pod{}
	_ = tfJobReconciler.Get(context.TODO(), worker.NamespacedName, pod)

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
			Name:  "tensorflow",
			State: containerState,
		},
	}
	pod.Status.Phase = status
	if status == corev1.PodRunning {
		now := metav1.Now()
		pod.Status.StartTime = &now
	}
	_ = tfJobReconciler.Status().Update(context.Background(), pod)
}

func createTFJob(jobName string, replicas int32) *tfv1.TFJob {
	successPolicy := tfv1.SuccessPolicyAllWorkers
	tfjob1 := &tfv1.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			UID:       "12345",
		},

		Spec: tfv1.TFJobSpec{
			TFReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
				"Worker": {
					Replicas:      &replicas,
					RestartPolicy: "Never",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "tensorflow",
									Image: "kubedl/tf-mnist-with-summaries:1.0",
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
	return tfjob1
}

// not used, maybe using later..
func createPodForJob(podName string, job *tfv1.TFJob) *corev1.Pod {
	labelGroupName := v1.GroupNameLabel
	labelJobName := v1.JobNameLabel
	groupName := tfv1.GroupName
	labels := map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(job.Name, "/", "-", -1),
	}
	labels[v1.ReplicaTypeLabel] = "Worker"
	labels[v1.ReplicaIndexLabel] = "0"
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              podName,
			Labels:            labels,
			Namespace:         "default",
			DeletionTimestamp: nil,
		},

		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "tensorflow",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 0,
						},
					},
				},
			},
			Phase: corev1.PodSucceeded,
		},
	}
}
