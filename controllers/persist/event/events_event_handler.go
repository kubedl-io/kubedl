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

	"github.com/alibaba/kubedl/controllers/persist/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueForEvent{}

// enqueueForEvent implements EventHandler interface and filter out events
// not provided by kubedl managed objects.
type enqueueForEvent struct {
	client client.Client
}

func (e *enqueueForEvent) Create(evt event.CreateEvent, queue workqueue.RateLimitingInterface) {
	evtObj := evt.Object.(*corev1.Event)
	if !e.isKubeDLManagedObject(evtObj.InvolvedObject) {
		return
	}
	queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: evtObj.Namespace,
		Name:      evtObj.Name,
	}})
}

func (e *enqueueForEvent) Update(evt event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	oldEvt := evt.ObjectOld.(*corev1.Event)
	newEvt := evt.ObjectNew.(*corev1.Event)
	if e.isKubeDLManagedObject(oldEvt.InvolvedObject) {
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: oldEvt.Namespace,
			Name:      oldEvt.Name,
		}})
	}
	if e.isKubeDLManagedObject(newEvt.InvolvedObject) {
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: newEvt.Namespace,
			Name:      newEvt.Name,
		}})
	}

}

func (e *enqueueForEvent) Delete(evt event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	evtObj := evt.Object.(*corev1.Event)
	if !e.isKubeDLManagedObject(evtObj.InvolvedObject) {
		return
	}
	queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: evtObj.Namespace,
		Name:      evtObj.Name,
	}})
}

func (e *enqueueForEvent) Generic(evt event.GenericEvent, queue workqueue.RateLimitingInterface) {
	e.Create(event.CreateEvent{Meta: evt.Meta, Object: evt.Object}, queue)
}

func (e *enqueueForEvent) isKubeDLManagedObject(ref corev1.ObjectReference) bool {
	if util.IsKubeDLManagedJobKind(ref.Kind) {
		return true
	}
	if ref.Kind == "Pod" && e.isKubeDLManagedPod(ref) {
		return true
	}
	return false
}

func (e *enqueueForEvent) isKubeDLManagedPod(ref corev1.ObjectReference) bool {
	pod := corev1.Pod{}
	if err := e.client.Get(context.Background(), types.NamespacedName{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}, &pod); err != nil {
		if errors.IsNotFound(err) {
			return true
		}
		return false
	}
	return util.IsKubeDLManagedPod(&pod)
}
