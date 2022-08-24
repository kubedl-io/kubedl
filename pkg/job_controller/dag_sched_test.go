package job_controller

import (
	"strings"
	"testing"
	"time"

	"github.com/alibaba/kubedl/apis"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestDAGConditions(t *testing.T) {
	testCases := []struct {
		name            string
		job             *trainingv1alpha1.TFJob
		pods            []*corev1.Pod
		inputConditions []v1.DAGCondition
		expectReady     bool
	}{
		{
			name:            "tf job with no dag conditions",
			job:             createTFJob("job-1", 3, 1, 0, 0),
			pods:            nil,
			inputConditions: nil,
			expectReady:     true,
		},
		{
			name:            "tf worker with dag conditions and ps without conditions, ps has not created yet but reconcile worker",
			job:             createTFJob("job-2", 3, 1, 0, 0),
			pods:            []*corev1.Pod{},
			inputConditions: []v1.DAGCondition{{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning}},
			expectReady:     false,
		},
		{
			name: "tf worker with dag conditions and ps without conditions, ps has just created but reconcile worker",
			job:  createTFJob("job-3", 3, 1, 0, 0),
			pods: []*corev1.Pod{
				createPodForJob("job-3-ps-0", "ps", "0", "job-3", corev1.PodPending),
			},
			inputConditions: []v1.DAGCondition{{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning}},
			expectReady:     false,
		},
		{
			name: "tf worker with dag conditions and ps without conditions, ps running and reconcile worker",
			job:  createTFJob("job-4", 3, 1, 0, 0),
			pods: []*corev1.Pod{
				createPodForJob("job-4-ps-0", "ps", "0", "job-4", corev1.PodRunning),
			},
			inputConditions: []v1.DAGCondition{{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning}},
			expectReady:     true,
		},
		{
			name: "tf worker has 2 dag conditions, both ps and chief [only for test], chief has not created yet",
			job:  createTFJob("job-5", 3, 1, 0, 1),
			pods: []*corev1.Pod{
				createPodForJob("job-5-ps-0", "ps", "0", "job-5", corev1.PodRunning),
			},
			inputConditions: []v1.DAGCondition{
				{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning},
				{Upstream: trainingv1alpha1.TFReplicaTypeChief, OnPhase: corev1.PodRunning},
			},
			expectReady: false,
		},
		{
			name: "tf worker has 2 dag conditions, both ps and chief [only for test], chief has just created",
			job:  createTFJob("job-6", 3, 1, 0, 1),
			pods: []*corev1.Pod{
				createPodForJob("job-6-ps-0", "ps", "0", "job-6", corev1.PodRunning),
				createPodForJob("job-6-chief-0", "chief", "0", "job-6", corev1.PodPending),
			},
			inputConditions: []v1.DAGCondition{
				{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning},
				{Upstream: trainingv1alpha1.TFReplicaTypeChief, OnPhase: corev1.PodRunning},
			},
			expectReady: false,
		},
		{
			name: "tf worker has 2 dag conditions, both ps and chief [only for test], all running",
			job:  createTFJob("job-7", 3, 1, 0, 1),
			pods: []*corev1.Pod{
				createPodForJob("job-7-ps-0", "ps", "0", "job-7", corev1.PodRunning),
				createPodForJob("job-7-chief-0", "chief", "0", "job-7", corev1.PodRunning),
			},
			inputConditions: []v1.DAGCondition{
				{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning},
				{Upstream: trainingv1alpha1.TFReplicaTypeChief, OnPhase: corev1.PodRunning},
			},
			expectReady: true,
		},
		{
			name: "tf worker with dag conditions and has multiple ps, ps not all running and reconcile worker",
			job:  createTFJob("job-10", 3, 2, 0, 0),
			pods: []*corev1.Pod{
				createPodForJob("job-10-ps-0", "ps", "0", "job-10", corev1.PodRunning),
			},
			inputConditions: []v1.DAGCondition{{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning}},
			expectReady:     false,
		},
		{
			name: "tf worker with dag conditions and has multiple ps, ps has been all running and reconcile worker",
			job:  createTFJob("job-11", 3, 2, 0, 0),
			pods: []*corev1.Pod{
				createPodForJob("job-11-ps-0", "ps", "0", "job-11", corev1.PodRunning),
				createPodForJob("job-11-ps-1", "ps", "0", "job-11", corev1.PodRunning),
			},
			inputConditions: []v1.DAGCondition{{Upstream: trainingv1alpha1.TFReplicaTypePS, OnPhase: corev1.PodRunning}},
			expectReady:     true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tc := testCase
			_ = apis.AddToScheme(scheme.Scheme)
			jc := JobController{}
			ready := jc.dagConditionsReady(tc.job, tc.job.Spec.TFReplicaSpecs, tc.pods, tc.inputConditions)
			if ready != tc.expectReady {
				t.Errorf("unexpected dag conditions ready, expected: %v, got: %v", tc.expectReady, ready)
			}
		})
	}
}
func createTFJob(jobName string, workerReplicas, psReplicas, aimasterReplicas, chiefReplicas int32) *trainingv1alpha1.TFJob {
	successPolicy := v1.SuccessPolicyAllWorkers
	demoReplicaSpec := func(replicas int32) *v1.ReplicaSpec {
		return &v1.ReplicaSpec{
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
		}
	}
	tfjob := &trainingv1alpha1.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			UID:       "12345",
		},
		Spec: trainingv1alpha1.TFJobSpec{
			TFReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
				"Worker": demoReplicaSpec(workerReplicas),
			},
			SuccessPolicy: &successPolicy,
		},
		Status: v1.JobStatus{},
	}
	if psReplicas > 0 {
		tfjob.Spec.TFReplicaSpecs["PS"] = demoReplicaSpec(psReplicas)
	}
	if aimasterReplicas > 0 {
		tfjob.Spec.TFReplicaSpecs["AIMaster"] = demoReplicaSpec(aimasterReplicas)
	}
	if chiefReplicas > 0 {
		tfjob.Spec.TFReplicaSpecs["Chief"] = demoReplicaSpec(chiefReplicas)
	}
	return tfjob
}
func createPodForJob(podName, rt, index, jobName string, phase corev1.PodPhase) *corev1.Pod {
	labelGroupName := v1.GroupNameLabel
	labelJobName := v1.JobNameLabel
	groupName := "training.kubedl.io"
	labels := map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(jobName, "/", "-", -1),
	}
	labels[v1.ReplicaTypeLabel] = rt
	labels[v1.ReplicaIndexLabel] = index
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              podName,
			Labels:            labels,
			Namespace:         "default",
			DeletionTimestamp: nil,
		},
		Status: corev1.PodStatus{Phase: phase},
	}
}

func NewFakeTFJobBuilder() *FakeTFJobBuilder {
	return &FakeTFJobBuilder{
		j: &trainingv1alpha1.TFJob{
			ObjectMeta: metav1.ObjectMeta{},

			Spec: trainingv1alpha1.TFJobSpec{
				TFReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{},
			},
			Status: v1.JobStatus{},
		},
	}
}

type FakeTFJobBuilder struct {
	j *trainingv1alpha1.TFJob
}

func (fb *FakeTFJobBuilder) WithName(name string) *FakeTFJobBuilder {
	fb.j.Name = name
	return fb
}

func (fb *FakeTFJobBuilder) WithReplicaSpec(replicas int32, rtype v1.ReplicaType, restartPolicy v1.RestartPolicy) *FakeTFJobBuilder {
	fb.j.Spec.TFReplicaSpecs[rtype] = &v1.ReplicaSpec{
		Replicas:      &replicas,
		RestartPolicy: restartPolicy,
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
	}

	return fb
}

func (fb *FakeTFJobBuilder) WithJobStatus(status v1.JobStatus) *FakeTFJobBuilder {
	fb.j.Status = status
	return fb
}

func (fb *FakeTFJobBuilder) Build() *trainingv1alpha1.TFJob {
	return fb.j
}

func createPodForJobWithStartTime(podName, rt, index, jobName string, phase corev1.PodPhase, starTime time.Time) *corev1.Pod {
	p := createPodForJob(podName, rt, index, jobName, phase)
	st := metav1.NewTime(starTime)
	p.Status.StartTime = &st
	return p
}
