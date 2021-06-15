package coscheduler

import (
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	testjobv1 "github.com/alibaba/kubedl/pkg/test_job/v1"
	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
)

var (
	controllerKind = testjobv1.SchemeGroupVersionKind
)

func TestCreateGang(t *testing.T) {
	testCases := []int32{
		1, 2, 3, 9,
	}

	for workerNumber := range testCases {
		testManager, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
		testScheduler := NewKubeCoscheduler(testManager)
		testJob := testutilv1.NewTestJob(workerNumber)

		testObject, _ := testScheduler.CreateGang(testJob, testJob.Spec.TestReplicaSpecs)
		testPodGroup := testObject.(*v1alpha1.PodGroup)
		assert.Equal(t, testPodGroup.Name, testutilv1.TestJobName)
		assert.Equal(t, testPodGroup.Namespace, metav1.NamespaceDefault)
		assert.Equal(t, testPodGroup.Spec.MinMember, int32(workerNumber))
	}
}

func TestBindPodToGang(t *testing.T) {
	testCases := []int32{
		1, 2, 3, 9,
	}

	for workerNumber := range testCases {
		testManager, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
		testScheduler := NewKubeCoscheduler(testManager)
		testJob := testutilv1.NewTestJob(workerNumber)

		testPodSpec := testutilv1.NewTestReplicaSpecTemplate()
		testPodSpec.Labels = make(map[string]string)
		testObject, _ := testScheduler.CreateGang(testJob, testJob.Spec.TestReplicaSpecs)
		testPodGroup := testObject.(*v1alpha1.PodGroup)

		testScheduler.BindPodToGang(&testPodSpec, testPodGroup)
		assert.Equal(t, testPodSpec.Labels["pod-group.scheduling.sigs.k8s.io"], testPodGroup.Name)

		assert.Equal(t, testPodSpec.Labels["pod-group.scheduling.sigs.k8s.io/name"], testPodGroup.Name)
		assert.Equal(t, testPodSpec.Labels["pod-group.scheduling.sigs.k8s.io/min-available"], string(testPodGroup.Spec.MinMember))
	}
}
