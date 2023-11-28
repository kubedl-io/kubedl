package job_controller

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	testjobv1 "github.com/alibaba/kubedl/pkg/test_job/v1"
	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
)

func TestSetRestartPolicy(t *testing.T) {
	type tc struct {
		testJob               *testjobv1.TestJob
		expectedRestartPolicy v1.RestartPolicy
		expectedType          apiv1.ReplicaType
	}
	testCase := []tc{
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyExitCode
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyNever
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyAlways
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyAlways,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyOnFailure
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyOnFailure,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
	}
	for _, c := range testCase {
		spec := c.testJob.Spec.TestReplicaSpecs[c.expectedType]
		podTemplate := spec.Template
		setRestartPolicy(&podTemplate, spec)
		if podTemplate.Spec.RestartPolicy != c.expectedRestartPolicy {
			t.Errorf("Expected %s, got %s", c.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
		}
	}
}

func TestGetPodSlices(t *testing.T) {
	testCases := []struct {
		replicas int
		pods     []*v1.Pod
		expected [][]*v1.Pod
	}{
		{
			replicas: 3,
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("pod-0", "worker", "0"),
				newPodWithReplicaTypeAndIndex("pod-1", "worker", "1"),
				newPodWithReplicaTypeAndIndex("pod-2", "worker", "2"),
			},
			expected: [][]*v1.Pod{
				{newPodWithReplicaTypeAndIndex("pod-0", "worker", "0")},
				{newPodWithReplicaTypeAndIndex("pod-1", "worker", "1")},
				{newPodWithReplicaTypeAndIndex("pod-2", "worker", "2")},
			},
		},
		{
			replicas: 3,
			pods:     []*v1.Pod{},
			expected: [][]*v1.Pod{
				nil,
				nil,
				nil,
			},
		},
		{
			replicas: 3,
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("pod-0", "worker", "0"),
				newPodWithReplicaTypeAndIndex("pod-2", "worker", "2"),
			},
			expected: [][]*v1.Pod{
				{newPodWithReplicaTypeAndIndex("pod-0", "worker", "0")},
				nil,
				{newPodWithReplicaTypeAndIndex("pod-2", "worker", "2")},
			},
		},
		{
			replicas: 3,
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("pod-0", "worker", "0"),
				newPodWithReplicaTypeAndIndex("pod-1", "worker", "1"),
				newPodWithReplicaTypeAndIndex("pod-2", "worker", "2"),
				newPodWithReplicaTypeAndIndex("pod-3", "worker", "3"),
				newPodWithReplicaTypeAndIndex("pod-4", "worker", "4"),
			},
			expected: [][]*v1.Pod{
				{newPodWithReplicaTypeAndIndex("pod-0", "worker", "0")},
				{newPodWithReplicaTypeAndIndex("pod-1", "worker", "1")},
				{newPodWithReplicaTypeAndIndex("pod-2", "worker", "2")},
				{newPodWithReplicaTypeAndIndex("pod-3", "worker", "3")},
				{newPodWithReplicaTypeAndIndex("pod-4", "worker", "4")},
			},
		},
	}

	for index, testCase := range testCases {
		jc := JobController{}

		podSlices := jc.GetPodSlices(testCase.pods, testCase.replicas, nil)
		if !reflect.DeepEqual(podSlices, testCase.expected) {
			t.Errorf("[%d]unexpected pod slices, expected: %+v, got: %+v", index, testCase.expected, podSlices)
		}
	}
}

func TestReconcilePods(t *testing.T) {
	testCases := []struct {
		name            string
		job             *testjobv1.TestJob
		pods            []*v1.Pod
		rtype           apiv1.ReplicaType
		expectedPodsNum int
	}{
		{
			name:            "scale out pods from empty to target replicas",
			job:             testutilv1.NewTestNamedJob("job-1", 2),
			pods:            []*v1.Pod{},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 2,
		},
		{
			name:            "scale out pods from not-satisfied-num to target replicas",
			job:             testutilv1.NewTestNamedJob("job-2", 2),
			pods:            []*v1.Pod{newPodWithReplicaTypeAndIndex("job-2-worker-0", "worker", "0")},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 2,
		},
		{
			name: "scale in pods from out-of-replicas to target replicas",
			job:  testutilv1.NewTestNamedJob("job-3", 2),
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("job-3-worker-0", "worker", "0"),
				newPodWithReplicaTypeAndIndex("job-3-worker-1", "worker", "1"),
				newPodWithReplicaTypeAndIndex("job-3-worker-2", "worker", "2"),
			},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 2,
		},
		{
			name: "delete failed pod with reason can be failover, Killed",
			job:  testutilv1.NewTestNamedJob("job-4", 2),
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("job-4-worker-0", "worker", "0"),
				newPodFailedWithReason("job-4-worker-1", "worker", "1", "Killed", 1),
			},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 1,
		},
		{
			name: "delete failed pod with reason can be failover, Evicted",
			job:  testutilv1.NewTestNamedJob("job-5", 2),
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("job-5-worker-0", "worker", "0"),
				newPodFailedWithReason("job-5-worker-1", "worker", "1", "Evicted", 1),
			},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 1,
		},
		{
			name: "delete failed pod with reason can not be failover, Unknown",
			job:  testutilv1.NewTestNamedJob("job-5", 2),
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("job-5-worker-0", "worker", "0"),
				newPodFailedWithReason("job-5-worker-1", "worker", "1", "Unknown", 1),
			},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 2,
		},
		{
			name: "delete failed pod with exitcode can be failover, 137",
			job:  testutilv1.NewTestNamedJob("job-5", 2),
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("job-5-worker-0", "worker", "0"),
				newPodFailedWithReason("job-5-worker-1", "worker", "1", "Failed", 137),
			},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 1,
		},
		{
			name: "delete failed pod with exitcode can not be failover, 1",
			job:  testutilv1.NewTestNamedJob("job-5", 2),
			pods: []*v1.Pod{
				newPodWithReplicaTypeAndIndex("job-5-worker-0", "worker", "0"),
				newPodFailedWithReason("job-5-worker-1", "worker", "1", "Failed", 1),
			},
			rtype:           testjobv1.TestReplicaTypeWorker,
			expectedPodsNum: 2,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			_ = testjobv1.AddToScheme(scheme.Scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			fakeRecorder := record.NewFakeRecorder(10)

			for _, p := range c.pods {
				_ = fakeClient.Create(context.Background(), p.DeepCopy())
				if p.Status.Phase != "" {
					_ = fakeClient.Update(context.Background(), p)
				}
			}

			jc := JobController{
				Expectations:   controller.NewControllerExpectations(),
				Recorder:       fakeRecorder,
				Controller:     &TestJobController{},
				Client:         fakeClient,
				PodControl:     NewPodControl(fakeClient, fakeRecorder),
				ServiceControl: NewServiceControl(fakeClient, fakeRecorder),
			}

			err := jc.ReconcilePods(context.Background(), c.job, &c.job.Status, c.pods, c.rtype, c.job.Spec.TestReplicaSpecs[c.rtype],
				c.job.Spec.TestReplicaSpecs, &apiv1.RunPolicy{}, pointer.BoolPtr(false))
			if err != nil {
				t.Errorf("failed to ReconcilePods, err: %v", err)
			}

			pods := v1.PodList{}
			err = fakeClient.List(context.Background(), &pods)
			if err != nil {
				t.Errorf("failed to list pods, err: %v", err)
			}

			if c.expectedPodsNum != len(pods.Items) {
				t.Errorf("unexpected pod num, expected: %v, actual: %v", c.expectedPodsNum, len(pods.Items))
			}
		})
	}
}

func newPodWithReplicaTypeAndIndex(name, rt, index string) *v1.Pod {
	p := newPod(name, v1.PodRunning)
	if p.Labels == nil {
		p.Labels = make(map[string]string)
	}
	p.Labels[apiv1.ReplicaTypeLabel] = rt
	p.Labels[apiv1.ReplicaIndexLabel] = index
	return p
}

func newPodFailedWithReason(name, rt, index, reason string, exitCode int) *v1.Pod {
	p := newPodWithReplicaTypeAndIndex(name, rt, index)
	p.Status.Phase = v1.PodFailed
	p.Status.Reason = reason
	if exitCode != 0 {
		p.Status.ContainerStatuses = []v1.ContainerStatus{
			{
				Name: p.Spec.Containers[0].Name,
				State: v1.ContainerState{
					Terminated: &v1.ContainerStateTerminated{
						ExitCode: int32(exitCode),
					},
				},
			},
		}
	}
	return p
}
