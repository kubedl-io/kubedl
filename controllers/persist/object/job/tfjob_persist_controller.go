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

	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
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
	jobPersistCtrlMap[&tfv1.TFJob{}] = NewTFJobPersistController
}

func NewTFJobPersistController(mgr ctrl.Manager, handler *jobPersistHandler) PersistController {
	return &TFJobPersistController{
		client:  mgr.GetClient(),
		handler: handler,
	}
}

var _ reconcile.Reconciler = &TFJobPersistController{}

type TFJobPersistController struct {
	client  ctrlruntime.Client
	handler *jobPersistHandler
}

func (pc *TFJobPersistController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Parse uid and object name from request.Name field.
	id, name, err := util.ParseIDName(req.Name)
	if err != nil {
		log.Error(err, "failed to parse request key")
		return ctrl.Result{}, err
	}

	tfJob := tfv1.TFJob{}
	err = pc.client.Get(context.Background(), types.NamespacedName{
		Namespace: req.Namespace,
		Name:      name,
	}, &tfJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to fetch tf job but it has been deleted.", "key", req.String())
			if err = pc.handler.Delete(tfJob.Namespace, tfJob.Name, id); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Persist tf job object into storage backend.
	if err = pc.handler.Save(&tfJob, tfJob.Kind, tfJob.Spec.TFReplicaSpecs, &tfJob.Status); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func (pc *TFJobPersistController) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("TFJobPersistController", mgr, controller.Options{Reconciler: pc})
	if err != nil {
		return err
	}

	// Watch events with event events-handler.
	if err = c.Watch(&source.Kind{Type: &tfv1.TFJob{}}, &enqueueForJob{}); err != nil {
		return err
	}
	return nil
}
