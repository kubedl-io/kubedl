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
	"reflect"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/pkg/cache_backend/registry"
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

// +kubebuilder:rbac:groups=cache.kubedl.io,resources=cachebackends,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.kubedl.io,resources=cachebackends/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *CacheBackendReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling for CacheBackend %s", req.Name))
	// Get cacheBackend
	cacheBackend := &cachev1alpha1.CacheBackend{}
	err := r.Get(context.Background(), req.NamespacedName, cacheBackend)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("CacheBackend doesn't exist", "name", req.String())
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "Failed to get CacheBackend")
		return reconcile.Result{}, err
	}

	status := cacheBackend.Status
	oldStatus := cacheBackend.Status.DeepCopy()

	if len(status.UsedBy) != 0 {
		status.LastUsedTime = nil
	}
	status.UsedNum = len(status.UsedBy)

	r.Log.Info(fmt.Sprintf("Current UsedNum equals %d, UsedBy %s", status.UsedNum, status.UsedBy))

	if status.CacheStatus == cachev1alpha1.PVCCreated && cacheBackend.Spec.Options.IdleTime != 0 && len(status.UsedBy) == 0 {
		idleTime := cacheBackend.Spec.Options.IdleTime
		if status.LastUsedTime == nil {
			status.UsedNum = len(status.UsedBy)
			status.LastUsedTime = &metav1.Time{Time: time.Now()}
			err = r.updateCacheBackendStatus(cacheBackend, &status)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			// it means that CacheBackend is into idle time
			return reconcile.Result{RequeueAfter: idleTime * time.Second}, nil
		} else {
			if time.Now().After(status.LastUsedTime.Add(idleTime * time.Second)) {
				r.Log.V(2).Info("cache backend has expired since last used, delete it",
					"name", cacheBackend.Name, "lastUsed", status.LastUsedTime)
				err = r.Delete(context.Background(), cacheBackend)
				return reconcile.Result{Requeue: err != nil}, err
			}
		}
	}

	// Check if pvc has created, if created, then update status to PVCCreated and return
	pvc := &v1.PersistentVolumeClaim{}
	pvcName := cacheBackend.Name
	err = r.Get(context.Background(), types.NamespacedName{
		Namespace: cacheBackend.Namespace,
		Name:      pvcName,
	}, pvc)

	if err == nil {
		r.Log.Info(fmt.Sprintf("PVC %s found", pvc.Name))
		status.CacheStatus = cachev1alpha1.PVCCreated
		err = r.updateCacheBackendStatus(cacheBackend, &status)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// If the pvc is not found, then there are two possible scenarios
	// 1) Cache job has already been committed and pvc is creating
	r.Log.Info(fmt.Sprintf("PVC %s not found, try to create CacheBackend", pvcName))
	if cacheBackend.Status.CacheStatus == cachev1alpha1.PVCCreating {
		r.Log.Info(fmt.Sprintf("PVC %s is creating", pvcName), "CacheBackend", cacheBackend.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	// 2) Cache job has not created, reconciler will start creating a cache job to generate the pvc
	if cacheBackend.Spec.CacheEngine == nil {
		r.Log.Error(err, " CacheEngine is undefined", "CacheBackend", cacheBackend.Name)
		return reconcile.Result{}, nil
	} else {
		// Different cache job are created based on the cacheEngine specified in cacheBackend.spec, which is pluggable
		cacheBackendName, err := registry.CacheBackendName(cacheBackend.Spec.CacheEngine)
		status.CacheEngine = cacheBackendName
		if err != nil {
			r.Log.Error(err, " Failed to get CacheBackend name in registry", "CacheBackend", cacheBackend.Name)
			return reconcile.Result{Requeue: true}, err
		}
		cacheEngine := registry.Get(cacheBackendName)
		err = cacheEngine.CreateCacheJob(cacheBackend)
		if err != nil {
			r.Log.Error(err, " Failed to create job with cache engine", "CacheBackend", cacheBackend.Name)
			// Update status
			status.CacheStatus = cachev1alpha1.CacheFailed
			err = r.updateCacheBackendStatus(cacheBackend, &status)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{Requeue: true}, err
		}

		// Update status
		status.CacheStatus = cachev1alpha1.PVCCreating
		status.LastUsedTime = nil
	}

	if !reflect.DeepEqual(oldStatus, status) {
		err := r.updateCacheBackendStatus(cacheBackend, &status)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *CacheBackendReconciler) updateCacheBackendStatus(cacheBackend *cachev1alpha1.CacheBackend,
	status *cachev1alpha1.CacheBackendStatus) error {

	cacheCopy := cacheBackend.DeepCopy()
	cacheCopy.Status = *status

	err := r.Status().Update(context.Background(), cacheCopy)
	if err != nil {
		r.Log.Error(err, "failed to update CacheBackend", "CacheBackend", cacheBackend.Name)
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
