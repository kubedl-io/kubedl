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

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/cache_backend/registry"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *CacheBackendReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	// Get cacheBackend
	cacheBackend := &cachev1alpha1.CacheBackend{}
	err := r.Get(context.Background(), req.NamespacedName, cacheBackend)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("cacheBackend doesn't exist", "name", req.String())
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "fail to get cacheBackend")
		return reconcile.Result{}, err
	}

	// When pvc has created, no need for reconcile
	if cacheBackend.Status.CacheStatus == cachev1alpha1.PVCCreated {
		r.Log.Info(fmt.Sprintf("cacheBackend status: is %s, pvc has created, skip reconcile", cacheBackend.Status.CacheStatus), "cacheBackend", cacheBackend.Name)
		return reconcile.Result{}, nil
	}

	// Check if pvc has created, if created, then update status to PVCCreated and return
	pvc := &v1.PersistentVolumeClaim{}
	pvcName := cacheBackend.Name
	err = r.Get(context.Background(), types.NamespacedName{
		Namespace: cacheBackend.Namespace,
		Name:      pvcName,
	}, pvc)

	if err == nil {
		r.Log.Info(fmt.Sprintf("pvc %s found", pvc.Name))
		err = r.updateCacheBackendStatus(cacheBackend, cachev1alpha1.PVCCreated)
		if err != nil {
			return ctrl.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// If the pvc is not found, then there are two possible scenarios
	// 1) Cache job has already been committed and pvc is creating
	r.Log.Info(fmt.Sprintf("pvc %s not found, try to create cache backend", pvcName))
	if cacheBackend.Status.CacheStatus == cachev1alpha1.PVCCreating {
		r.Log.Info(fmt.Sprintf("pvc %s is creating", pvcName), "cacheBackend", cacheBackend.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	// 2) Cache job has not create, reconciler will start creating a cache job to generate the pvc
	if cacheBackend.Spec.CacheEngine == nil {
		r.Log.Error(err, "cacheEngine is undefined", "cache backend", cacheBackend.Name)
		return reconcile.Result{}, nil
	} else {
		// Different cache job are created based on the cacheEngine specified in cacheBackend.spec, which is pluggable
		cacheBackendName, err := registry.CacheBackendName(cacheBackend.Spec.CacheEngine)
		if err != nil {
			r.Log.Error(err, "failed to get cache backend name in registry", "cacheBackend", cacheBackend.Name)
			return ctrl.Result{}, err
		}
		cacheEngine := registry.Get(cacheBackendName)
		err = cacheEngine.CreateCacheJob(cacheBackend)
		if err != nil {
			r.Log.Error(err, "failed to create job with cache engine", "cacheBackend", cacheBackend.Name)
			// Update status
			err = r.updateCacheBackendStatus(cacheBackend, cachev1alpha1.CacheFailed)
			if err != nil {
				return ctrl.Result{}, err
			}
			return reconcile.Result{Requeue: true}, err
		}

		// Update status
		err = r.updateCacheBackendStatus(cacheBackend, cachev1alpha1.PVCCreating)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *CacheBackendReconciler) updateCacheBackendStatus(cacheBackend *cachev1alpha1.CacheBackend, status cachev1alpha1.CacheStatus) error {
	cacheBackendStatus := cacheBackend.Status.DeepCopy()
	cacheCopy := cacheBackend.DeepCopy()
	cacheCopy.Status = *cacheBackendStatus
	cacheCopy.Status.CacheStatus = status
	err := r.Status().Update(context.Background(), cacheCopy)
	if err != nil {
		r.Log.Error(err, "failed to update cacheBackend", "cacheBackend", cacheBackend.Name)
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CacheBackendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.CacheBackend{}).
		Complete(r)
}
