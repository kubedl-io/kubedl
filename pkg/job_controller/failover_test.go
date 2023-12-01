package job_controller

import (
	"context"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kruisev1alpha1 "github.com/openkruise/kruise/apis/apps/v1alpha1"

	"github.com/alibaba/kubedl/apis"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func TestDoFailOverByAction(t *testing.T) {
	testCases := []struct {
		name         string
		pods         []client.Object
		failoverPods []*v1.Pod
		expectedPods int
		expectedCrr  int
		job          *trainingv1alpha1.TFJob
		rtype        apiv1.ReplicaType
		action       FailOverAction
	}{
		{
			name: "failover recreate some worker pods with restartPolicy=ExitCode",
			pods: []client.Object{
				createPodForJob("pod-0", "worker", "0", "job-1", v1.PodRunning),
				createPodForJob("pod-1", "worker", "1", "job-1", v1.PodRunning),
				createPodForJob("pod-2", "worker", "2", "job-1", v1.PodRunning),
				createPodForJob("pod-3", "worker", "3", "job-1", v1.PodRunning),
			},
			failoverPods: []*v1.Pod{
				createPodForJob("pod-1", "worker", "1", "job-1", v1.PodRunning),
				createPodForJob("pod-2", "worker", "2", "job-1", v1.PodRunning),
			},
			job: NewFakeTFJobBuilder().
				WithName("job-1").
				WithReplicaSpec(4, trainingv1alpha1.TFReplicaTypeWorker, apiv1.RestartPolicyExitCode).
				Build(),
			expectedPods: 2,
			rtype:        trainingv1alpha1.TFReplicaTypeWorker,
			action:       FailOverRecreate,
		},
		{
			name: "failover recreate all worker pods with restartPolicy=ExitCode",
			pods: []client.Object{
				createPodForJob("pod-0", "worker", "0", "job-1", v1.PodRunning),
				createPodForJob("pod-1", "worker", "1", "job-1", v1.PodRunning),
				createPodForJob("pod-2", "worker", "2", "job-1", v1.PodRunning),
				createPodForJob("pod-3", "worker", "3", "job-1", v1.PodRunning),
			},
			failoverPods: []*v1.Pod{
				createPodForJob("pod-0", "worker", "0", "job-1", v1.PodRunning),
				createPodForJob("pod-1", "worker", "1", "job-1", v1.PodRunning),
				createPodForJob("pod-2", "worker", "2", "job-1", v1.PodRunning),
				createPodForJob("pod-3", "worker", "3", "job-1", v1.PodRunning),
			},
			job: NewFakeTFJobBuilder().
				WithName("job-1").
				WithReplicaSpec(4, trainingv1alpha1.TFReplicaTypeWorker, apiv1.RestartPolicyExitCode).
				Build(),
			expectedPods: 0,
			rtype:        trainingv1alpha1.TFReplicaTypeWorker,
			action:       FailOverRecreate,
		},
		{
			name: "failover inplace restart some worker pods with restartPolicy=ExitCodeThenRestart",
			pods: []client.Object{
				createPodForJobWithStartTime("pod-0", "worker", "0", "job-1", v1.PodRunning, time.Now().Add(-time.Hour)),
				createPodForJobWithStartTime("pod-1", "worker", "1", "job-1", v1.PodRunning, time.Now().Add(-time.Hour)),
				createPodForJobWithStartTime("pod-2", "worker", "2", "job-1", v1.PodRunning, time.Now().Add(-time.Hour)),
				createPodForJobWithStartTime("pod-3", "worker", "3", "job-1", v1.PodRunning, time.Now().Add(-time.Hour)),
			},
			failoverPods: []*v1.Pod{
				createPodForJobWithStartTime("pod-0", "worker", "0", "job-1", v1.PodRunning, time.Now().Add(-time.Hour)),
				createPodForJobWithStartTime("pod-1", "worker", "1", "job-1", v1.PodRunning, time.Now().Add(-time.Hour)),
			},
			job: NewFakeTFJobBuilder().
				WithName("job-1").
				WithReplicaSpec(4, trainingv1alpha1.TFReplicaTypeWorker, apiv1.RestartPolicyExitCode).
				Build(),
			expectedPods: 4,
			expectedCrr:  2,
			rtype:        trainingv1alpha1.TFReplicaTypeWorker,
			action:       FailOverInPlaceRestart,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1.AddToScheme(scheme)
			_ = apis.AddToScheme(scheme)
			_ = kruisev1alpha1.AddToScheme(scheme)
			fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testCase.pods...).Build()
			fr := record.NewFakeRecorder(10)
			jc := JobController{
				Client:     fc,
				Recorder:   fr,
				PodControl: NewPodControl(fc, fr),
			}

			err := jc.DoFailOverByAction(testCase.job, testCase.failoverPods, testCase.action)
			if err != nil {
				t.Errorf("failed to do failover, err: %v", err)
			}

			if testCase.expectedCrr > 0 {
				crrs := kruisev1alpha1.ContainerRecreateRequestList{}
				err = fc.List(context.Background(), &crrs, client.InNamespace("default"))
				if err != nil {
					t.Error(err)
				}
				if len(crrs.Items) != testCase.expectedCrr {
					t.Errorf("unexpected crr size, expected: %v, got: %v", testCase.expectedCrr, len(crrs.Items))
				}
			}

			pods := v1.PodList{}
			if err = fc.List(context.Background(), &pods, client.InNamespace("default")); err != nil {
				t.Error(err)
			}
			if len(pods.Items) != testCase.expectedPods {
				t.Errorf("unexpected pods size, expected: %v, got: %v", testCase.expectedPods, len(pods.Items))
			}
		})
	}
}
