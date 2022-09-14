package controllers

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cacheBackend).Build()

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

func TestCachePolicyBasedOnIdleTime(t *testing.T) {
	testCases := []struct {
		testName      string
		cacheBackend  *v1alpha1.CacheBackend
		cacheStatus   v1alpha1.CacheStatus
		usedBy        []string
		expectDeleted bool
	}{
		{
			testName:      "fluid",
			cacheBackend:  testcase.NewFluidCacheBackend("testCacheBackend", "default"),
			cacheStatus:   v1alpha1.PVCCreated,
			usedBy:        []string{},
			expectDeleted: true,
		},
		{
			testName:      "fluid",
			cacheBackend:  testcase.NewFluidCacheBackend("testCacheBackend", "default"),
			cacheStatus:   v1alpha1.PVCCreating,
			usedBy:        []string{},
			expectDeleted: false,
		},
		{
			testName:      "fluid",
			cacheBackend:  testcase.NewFluidCacheBackend("testCacheBackend", "default"),
			cacheStatus:   v1alpha1.PVCCreated,
			usedBy:        []string{"test-job-1", "test-job-2"},
			expectDeleted: false,
		},
	}

	for _, testCase := range testCases {

		scheme := runtime.NewScheme()
		_ = apis.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)

		cacheBackend := testCase.cacheBackend
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cacheBackend).Build()
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

		// 1. At first, CacheBackend can be got successfully
		err := cacheReconciler.Get(context.TODO(), types.NamespacedName{
			Namespace: cacheBackend.Namespace,
			Name:      cacheBackend.Name,
		}, cacheBackend)
		assert.NoError(t, err)

		// 2. Update the IdleTime to control the longest time of CacheBackend to remain
		cacheBackend.Status.LastUsedTime = &metav1.Time{Time: time.Date(2022, 9, 1, 10, 39, 0, 0, time.UTC)}
		cacheBackend.Spec.Options.IdleTime = time.Now().Sub(cacheBackend.Status.LastUsedTime.Time)
		cacheBackend.Status.CacheStatus = testCase.cacheStatus
		cacheBackend.Status.UsedBy = testCase.usedBy

		err = cacheReconciler.Update(context.Background(), cacheBackend)
		assert.NoError(t, err)

		err = cacheReconciler.Status().Update(context.Background(), cacheBackend)
		assert.NoError(t, err)

		// 3. Controller will delete the infrequently used CacheBackend
		_, _ = cacheReconciler.Reconcile(context.Background(), request)

		err = cacheReconciler.Get(context.TODO(), types.NamespacedName{
			Namespace: cacheBackend.Namespace,
			Name:      cacheBackend.Name,
		}, cacheBackend)
		isDeleted := err != nil

		assert.Equal(t, testCase.expectDeleted, isDeleted)
	}
}
