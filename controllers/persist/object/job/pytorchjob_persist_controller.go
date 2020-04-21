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

package job

import (
	"context"

	v1 "github.com/alibaba/kubedl/api/pytorch/v1"
	"github.com/alibaba/kubedl/controllers/persist/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func init() {
	jobPersistCtrlMap[&v1.PyTorchJob{}] = NewPytorchJobPersistController
}

func NewPytorchJobPersistController(mgr ctrl.Manager, handler *jobPersistHandler) PersistController {
	return &PytorchJobPersistController{
		client:  mgr.GetClient(),
		handler: handler,
	}
}

var _ reconcile.Reconciler = &PytorchJobPersistController{}

type PytorchJobPersistController struct {
	client  ctrlruntime.Client
	handler *jobPersistHandler
}

func (pc *PytorchJobPersistController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Parse uid and object name from request.Name field.
	id, name, err := util.ParseIDName(req.Name)
	if err != nil {
		log.Error(err, "failed to parse request key")
		return ctrl.Result{}, err
	}

	pytorchJob := v1.PyTorchJob{}
	err = pc.client.Get(context.Background(), types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}, &pytorchJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to fetch pytorch job but it has been deleted.", "key", req.String())
			if err = pc.handler.Delete(pytorchJob.Namespace, pytorchJob.Name, id); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Persist pytorch job object into storage backend.
	if err = pc.handler.Save(&pytorchJob, pytorchJob.Kind, pytorchJob.Spec.PyTorchReplicaSpecs, &pytorchJob.Status); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func (pc *PytorchJobPersistController) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("PytorchJobPersistController", mgr, controller.Options{Reconciler: pc})
	if err != nil {
		return err
	}

	// Watch events with event events-handler.
	if err = c.Watch(&source.Kind{Type: &v1.PyTorchJob{}}, &enqueueForJob{}); err != nil {
		return err
	}
	return nil
}
