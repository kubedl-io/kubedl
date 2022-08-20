package fluid

import (
	"context"
	"testing"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/alibaba/kubedl/apis"
	testcase "github.com/alibaba/kubedl/pkg/cache_backend/test"
)

func TestCreateCacheJob(t *testing.T) {

	scheme := runtime.NewScheme()
	_ = apis.AddToScheme(scheme)

	cacheBackend := testcase.NewFluidCacheBackend("testCacheBackend", "jobName")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cacheBackend).Build()
	testFluidCache := &Cache{client: fakeClient}

	// create job
	_ = testFluidCache.CreateCacheJob(cacheBackend)

	// check if alluxio runtime created
	alluxioRuntime := &fluidv1alpha1.AlluxioRuntime{}
	_ = testFluidCache.client.Get(context.TODO(), types.NamespacedName{
		Namespace: cacheBackend.Namespace,
		Name:      cacheBackend.Name,
	}, alluxioRuntime)

	assert.Equal(t, "testCacheBackend", alluxioRuntime.Name)

	// check if dataset created
	dataset := &fluidv1alpha1.Dataset{}
	_ = testFluidCache.client.Get(context.TODO(), types.NamespacedName{
		Namespace: cacheBackend.Namespace,
		Name:      cacheBackend.Name,
	}, dataset)

	assert.Equal(t, "testCacheBackend", dataset.Name)

}
