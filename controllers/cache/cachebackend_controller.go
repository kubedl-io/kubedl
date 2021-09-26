/*
Copyright 2021 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/alibaba/kubedl/cmd/options"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	cachefactory "github.com/alibaba/kubedl/controllers/cache/factory"
)

func NewCacheBackendController(mgr ctrl.Manager, _ options.JobControllerConfiguration) *CacheBackendReconciler {
	return &CacheBackendReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("CacheBackend"),
		Scheme: mgr.GetScheme(),
	}
}

// CacheBackendReconciler reconciles a CacheBackend object
type CacheBackendReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cache.kubedl.io,resources=cachebackends,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.kubedl.io,resources=cachebackends/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CacheBackend object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.6.4/pkg/reconcile
func (r *CacheBackendReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	cacheBackend := &cachev1alpha1.CacheBackend{}

	// check if the cache backend is created
	err := r.Get(context.Background(), req.NamespacedName, cacheBackend)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("cacheBackend doesn't exist", "name", req.String())
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "fail to get cache backend")
		return reconcile.Result{}, err
	}

	// cache finished or failed
	if cacheBackend.Status.CacheStatus == cachev1alpha1.CacheSucceeded ||
		cacheBackend.Status.CacheStatus == cachev1alpha1.CacheFailed {
		r.Log.Info(fmt.Sprintf("cache status %s", cacheBackend.Status.CacheStatus), "cacheBackend", cacheBackend.Name)
		return reconcile.Result{}, nil
	}

	// submit cachebackend.spec.cacheEntity to cache entity and get pvc
	pvc := &v1.PersistentVolumeClaim{}
	if cacheBackend.Spec.CacheEngine == nil {
		r.Log.Error(err, "cacheEngine is undefined", "cache backend", cacheBackend.Name)
		return reconcile.Result{}, nil
	} else {
		err = r.submitCacheJobAndGetPVC(cacheBackend, pvc)
		if err != nil {
			r.Log.Error(err, "failed to create pvc for cacheBackend", "cacheBackend", cacheBackend.Name)
			return reconcile.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *CacheBackendReconciler) submitCacheJobAndGetPVC(cacheBackend *cachev1alpha1.CacheBackend,
	pvc *v1.PersistentVolumeClaim) (error error) {
	pvcName := cacheBackend.Name
	err := r.Get(context.Background(), types.NamespacedName{Namespace: cacheBackend.Namespace, Name: pvcName}, pvc)
	if err != nil {
		if errors.IsNotFound(err) {
			cacheProvider := cachefactory.GetCacheProvider(cacheBackend.Spec.CacheEngine)
			pvc = cacheProvider.CreateCacheJob(cacheBackend, pvcName)
		} else {
			return err
		}
	}
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *CacheBackendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.CacheBackend{}).
		Complete(r)
}
