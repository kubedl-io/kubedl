package patch

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	tfv1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func TestPatchBodyGeneration(t *testing.T) {
	patchReq := NewStrategicPatch().
		AddFinalizer("new-finalizer").
		RemoveFinalizer("old-finalizer").
		InsertLabel("new-label", "foo1").
		DeleteLabel("old-label").
		InsertAnnotation("new-annotation", "foo2").
		DeleteAnnotation("old-annotation")

	expectedPatchBody := `{"metadata":{"labels":{"new-label":"foo1","old-label":null},"annotations":{"new-annotation":"foo2","old-annotation":null},"finalizers":["new-finalizer"],"$deleteFromPrimitiveList/finalizers":["old-finalizer"]}}`

	if !reflect.DeepEqual(patchReq.String(), expectedPatchBody) {
		t.Fatalf("Not equal: \n%s \n%s", expectedPatchBody, patchReq.String())
	}

}

func TestPodPatchOperations(t *testing.T) {
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "main",
					Image: "busybox",
				},
			},
		},
	}
	nsName := types.NamespacedName{Name: "hello", Namespace: "default"}

	fake := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	_ = fake.Create(context.Background(), p)

	patch := NewStrategicPatch()
	patch.InsertLabel("foo", "bar")
	err := fake.Patch(context.Background(), p, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, p)
	assert.Nil(t, err)
	assert.Equal(t, p.Labels["foo"], "bar")
	patch.InsertLabel("foo", "bar1")
	err = fake.Patch(context.Background(), p, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, p)
	assert.Nil(t, err)
	assert.Equal(t, p.Labels["foo"], "bar1")

	patch = NewStrategicPatch()
	patch.InsertAnnotation("foo", "bar")
	err = fake.Patch(context.Background(), p, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, p)
	assert.Nil(t, err)
	assert.Equal(t, p.Annotations["foo"], "bar")

	patch = NewStrategicPatch()
	patch.AddFinalizer("finalizer")
	err = fake.Patch(context.Background(), p, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, p)
	assert.Nil(t, err)
	assert.Equal(t, len(p.Finalizers), 1)
	assert.Equal(t, p.Finalizers[0], "finalizer")
}

func TestJobPatchOperations(t *testing.T) {
	job := &tfv1.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: "default",
		},
		Spec: tfv1.TFJobSpec{
			TFReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
				tfv1.TFReplicaTypeWorker: {
					Replicas: pointer.Int32Ptr(1),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "main",
									Image: "busybox",
								},
							},
						},
					},
				},
			},
		},
	}
	nsName := types.NamespacedName{Name: "hello", Namespace: "default"}

	_ = tfv1.SchemeBuilder.AddToScheme(scheme.Scheme)
	fake := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	_ = fake.Create(context.Background(), job)

	patch := NewMergePatch()
	patch.InsertLabel("foo", "bar")
	err := fake.Patch(context.Background(), job, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, job)
	assert.Nil(t, err)
	assert.Equal(t, job.Labels["foo"], "bar")
	patch.InsertLabel("foo", "bar1")
	err = fake.Patch(context.Background(), job, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, job)
	assert.Nil(t, err)
	assert.Equal(t, job.Labels["foo"], "bar1")

	patch = NewMergePatch()
	patch.InsertAnnotation("foo", "bar")
	err = fake.Patch(context.Background(), job, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, job)
	assert.Nil(t, err)
	assert.Equal(t, job.Annotations["foo"], "bar")

	patch = NewMergePatch()
	patch.AddFinalizer("finalizer")
	err = fake.Patch(context.Background(), job, patch)
	assert.Nil(t, err)
	err = fake.Get(context.Background(), nsName, job)
	assert.Nil(t, err)
	assert.Equal(t, len(job.Finalizers), 0)
}
