package batch_scheduler

import (
	"testing"

	"github.com/kubernetes-sigs/kube-batch/pkg/apis/scheduling/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	testjobv1 "github.com/alibaba/kubedl/pkg/test_job/v1"
	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
)

func TestCreateGang(t *testing.T) {
	testCases := []int{
		1, 2, 3, 9,
	}

	for _, workerNumber := range testCases {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = testjobv1.AddToScheme(scheme)
		testJob := testutilv1.NewTestJob(workerNumber)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(testJob).Build()
		testScheduler := &kubeBatchScheduler{client: fakeClient}

		testObject, _ := testScheduler.CreateGang(testJob, testJob.Spec.TestReplicaSpecs, testJob.Spec.RunPolicy.SchedulingPolicy)
		testPodGroup := testObject.(*v1alpha1.PodGroup)
		assert.Equal(t, testPodGroup.Name, testutilv1.TestJobName)
		assert.Equal(t, testPodGroup.Namespace, metav1.NamespaceDefault)
		assert.Equal(t, testPodGroup.Spec.MinMember, int32(workerNumber))
	}
}

func TestBindPodToGang(t *testing.T) {
	testCases := []int{
		1, 2, 3, 9,
	}

	for _, workerNumber := range testCases {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = testjobv1.AddToScheme(scheme)
		testJob := testutilv1.NewTestJob(workerNumber)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(testJob).Build()
		testScheduler := &kubeBatchScheduler{client: fakeClient}

		testObject, _ := testScheduler.CreateGang(testJob, testJob.Spec.TestReplicaSpecs, testJob.Spec.RunPolicy.SchedulingPolicy)
		testPodGroup := testObject.(*v1alpha1.PodGroup)
		testPodSpec := testutilv1.NewTestReplicaSpecTemplate()
		err := testScheduler.BindPodToGang(testJob, &testPodSpec, testPodGroup, "")
		assert.NoError(t, err)
		assert.Equal(t, testPodSpec.Annotations[v1alpha1.GroupNameAnnotationKey], testPodGroup.Name)
	}
}
