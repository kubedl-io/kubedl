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

package event

import (
	"context"
	stderrors "errors"
	"fmt"
	"github.com/alibaba/kubedl/pkg/storage/backends/registry"

	"github.com/alibaba/kubedl/pkg/storage/backends"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "EventPersistController"
)

var log = logf.Log.WithName("event-persist-controller")

func NewEventPersistController(mgr ctrl.Manager, eventStorage string, region string) (*EventPersistController, error) {
	if eventStorage == "" {
		return nil, stderrors.New("empty event storage backend name")
	}

	// Get event storage backend from backends registry.
	eventBackend := registry.GetEventBackend(eventStorage)
	if eventBackend == nil {
		return nil, fmt.Errorf("event storage backend [%s] has not registered", eventStorage)
	}

	// Initialize event storage backend before event-persist-controller created.
	if err := eventBackend.Initialize(); err != nil {
		return nil, err
	}

	return &EventPersistController{
		region:       region,
		client:       mgr.GetClient(),
		eventBackend: eventBackend,
	}, nil
}

var _ reconcile.Reconciler = &EventPersistController{}

type EventPersistController struct {
	region       string
	client       ctrlruntime.Client
	eventBackend backends.EventStorageBackend
}

func (pc *EventPersistController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	event := corev1.Event{}
	err := pc.client.Get(context.Background(), req.NamespacedName, &event)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to fetch event but it has been deleted.", "key", req.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Persist event object into storage backend.
	if err = pc.eventBackend.SaveEvent(event.DeepCopy(), pc.region); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (pc *EventPersistController) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: pc})
	if err != nil {
		return err
	}

	// Watch events with event events-handler.
	if err = c.Watch(&source.Kind{Type: &corev1.Event{}}, &enqueueForEvent{client: mgr.GetClient()}); err != nil {
		return err
	}
	return nil
}
