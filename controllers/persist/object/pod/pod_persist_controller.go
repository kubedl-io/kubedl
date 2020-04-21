/*
Copyright 2020 The Alibaba Authors.

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

package pod

import (
	"context"
	stderrors "errors"
	"fmt"

	pytorchv1 "github.com/alibaba/kubedl/api/pytorch/v1"
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	xgboostv1alpha1 "github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	persistutil "github.com/alibaba/kubedl/controllers/persist/util"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/backends/registry"
	"github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "PodPersistController"
)

var log = logf.Log.WithName("pod-persist-controller")

func NewPodPersistController(mgr ctrl.Manager, podStorage string, region string) (*PodPersistController, error) {
	if podStorage == "" {
		return nil, stderrors.New("empty pod storage backend name")
	}

	// Get pod storage backend from backends registry.
	podBackend := registry.GetObjectBackend(podStorage)
	if podBackend == nil {
		return nil, fmt.Errorf("pod storage backend [%s] has not registered", podStorage)
	}

	// Initialize pod storage backend before pod-persist-controller created.
	if err := podBackend.Initialize(); err != nil {
		return nil, err
	}

	return &PodPersistController{
		region:     region,
		client:     mgr.GetClient(),
		podBackend: podBackend,
	}, nil
}

var _ reconcile.Reconciler = &PodPersistController{}

type PodPersistController struct {
	region     string
	client     ctrlruntime.Client
	podBackend backends.ObjectStorageBackend
}

func (pc *PodPersistController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Parse uid and object name from request.Name field.
	id, name, err := persistutil.ParseIDName(req.Name)
	if err != nil {
		log.Error(err, "failed to parse request key")
		return ctrl.Result{}, err
	}

	pod := &corev1.Pod{}
	err = pc.client.Get(context.Background(), types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to fetch pod but it has been deleted.", "key", req.String())
			if err = pc.podBackend.StopPod(name, req.Namespace, id); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	owner := util.GetControllerOwnerReference(pod.OwnerReferences)
	defaultContainerName := matchDefaultContainerName(owner.Kind)

	if err = pc.podBackend.SavePod(pod, defaultContainerName, pc.region); err != nil {
		log.Error(err, "error when persist pod object to storage backend", "pod", req.String())
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (pc *PodPersistController) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: pc})
	if err != nil {
		return err
	}

	// Watch events with pod events-handler.
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &enqueueForPod{}); err != nil {
		return err
	}
	return nil
}

func matchDefaultContainerName(kind string) string {
	switch kind {
	case tfv1.Kind:
		return tfv1.DefaultContainerName
	case pytorchv1.Kind:
		return pytorchv1.DefaultContainerName
	case xdlv1alpha1.Kind:
		return xdlv1alpha1.DefaultContainerName
	case xgboostv1alpha1.Kind:
		return xgboostv1alpha1.DefaultContainerName
	}
	return ""
}
