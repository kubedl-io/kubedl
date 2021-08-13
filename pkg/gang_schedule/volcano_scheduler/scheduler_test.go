package volcano_scheduler

import (
	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	testjobv1 "github.com/alibaba/kubedl/pkg/test_job/v1"
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
		fakeClient := fake.NewFakeClientWithScheme(scheme, testJob)
		testScheduler := &volcanoScheduler{client: fakeClient}

		testObject, _ := testScheduler.CreateGang(testJob, testJob.Spec.TestReplicaSpecs)
		testPodGroup := testObject.(*v1beta1.PodGroup)
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
		fakeClient := fake.NewFakeClientWithScheme(scheme, testJob)
		testScheduler := &volcanoScheduler{client: fakeClient}

		testObject, _ := testScheduler.CreateGang(testJob, testJob.Spec.TestReplicaSpecs)
		testPodGroup := testObject.(*v1beta1.PodGroup)
		testPodSpec := testutilv1.NewTestReplicaSpecTemplate()
		testScheduler.BindPodToGang(&testPodSpec, testPodGroup)
		assert.Equal(t, testPodSpec.Annotations[v1beta1.KubeGroupNameAnnotationKey], testPodGroup.Name)
	}
}
