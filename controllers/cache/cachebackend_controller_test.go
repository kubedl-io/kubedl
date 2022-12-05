package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/alibaba/kubedl/apis"
	"github.com/alibaba/kubedl/apis/cache/v1alpha1"
	cacheregistry "github.com/alibaba/kubedl/pkg/cache_backend/registry"
	testcase "github.com/alibaba/kubedl/pkg/cache_backend/test"
)

func TestCacheBackendStatus(t *testing.T) {

	// Can add other types of cache backends as test cases
	testCases := []v1alpha1.CacheBackend{
		*testcase.NewFluidCacheBackend("testCacheBackend", "default"),
	}

	for _, testCase := range testCases {
		scheme := runtime.NewScheme()
		_ = apis.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)

		cacheBackend := &testCase

		fakeClient := fake.NewFakeClientWithScheme(scheme, cacheBackend)

		// register cacheBackend
		cacheregistry.RegisterCacheBackends(fakeClient)

		cacheReconciler := &CacheBackendReconciler{
			Client: fakeClient,
			Log:    ctrl.Log.WithName("controllers").WithName("CacheBackend"),
			Scheme: scheme,
		}

		request := reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: cacheBackend.Namespace,
			Name:      cacheBackend.Name,
		}}

		_, _ = cacheReconciler.Reconcile(context.Background(), request)

		_ = cacheReconciler.Get(context.TODO(), types.NamespacedName{
			Namespace: cacheBackend.Namespace,
			Name:      cacheBackend.Name,
		}, cacheBackend)

		assert.Equal(t, "testCacheBackend", cacheBackend.Name)

		// When pvc has created, cacheBackend should update status to PVCCreated
		_ = cacheReconciler.Get(context.TODO(), types.NamespacedName{
			Namespace: cacheBackend.Namespace,
			Name:      cacheBackend.Name,
		}, cacheBackend)
		assert.Equal(t, v1alpha1.PVCCreating, cacheBackend.Status.CacheStatus)

		// Manually create pvc
		pvc := &corev1.PersistentVolumeClaim{}
		pvc.Namespace = cacheBackend.Namespace
		pvc.Name = cacheBackend.Name
		_ = cacheReconciler.Create(context.TODO(), pvc)

		// Check pvc has created
		_ = cacheReconciler.Get(context.TODO(), types.NamespacedName{
			Namespace: cacheBackend.Namespace,
			Name:      cacheBackend.Name,
		}, pvc)
		assert.Equal(t, cacheBackend.Name, pvc.Name)

		// Update status
		_, _ = cacheReconciler.Reconcile(context.Background(), request)
		_ = cacheReconciler.Get(context.TODO(), types.NamespacedName{
			Namespace: cacheBackend.Namespace,
			Name:      cacheBackend.Name,
		}, cacheBackend)
		assert.Equal(t, v1alpha1.PVCCreated, cacheBackend.Status.CacheStatus)
	}
}
