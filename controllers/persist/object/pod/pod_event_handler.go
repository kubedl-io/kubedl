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
	v1 "k8s.io/api/core/v1"

	"github.com/alibaba/kubedl/controllers/persist/util"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueForPod{}

// enqueueForPod implements EventHandler interface and filter out
// events of pod which are not controlled by kubedl jobs.
//
// Hack(SimonCqk): when controller watched a event and does pod persistency,
// we'd record pod's UID as well because when some pod deleted by kubedl and
// recreated, the later-coming pod will override the persisted one in storage
// backend, that's what we unexpected.
type enqueueForPod struct{}

func (e *enqueueForPod) Create(evt event.CreateEvent, queue workqueue.RateLimitingInterface) {
	if util.IsKubeDLManagedPod(evt.Object.(*v1.Pod)) {
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: evt.Object.GetNamespace(),
			Name:      util.IDName(evt.Object),
		}})
	}
}

func (e *enqueueForPod) Update(evt event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	if util.IsKubeDLManagedPod(evt.ObjectOld.(*v1.Pod)) {
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: evt.ObjectOld.GetNamespace(),
			Name:      util.IDName(evt.ObjectOld),
		}})
	}

	if util.IsKubeDLManagedPod(evt.ObjectNew.(*v1.Pod)) {
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: evt.ObjectNew.GetNamespace(),
			Name:      util.IDName(evt.ObjectNew),
		}})
	}
}

func (e *enqueueForPod) Delete(evt event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	if util.IsKubeDLManagedPod(evt.Object.(*v1.Pod)) {
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: evt.Object.GetNamespace(),
			Name:      util.IDName(evt.Object),
		}})
	}
}

func (e *enqueueForPod) Generic(evt event.GenericEvent, queue workqueue.RateLimitingInterface) {
	if util.IsKubeDLManagedPod(evt.Object.(*v1.Pod)) {
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: evt.Object.GetNamespace(),
			Name:      util.IDName(evt.Object),
		}})
	}
}
