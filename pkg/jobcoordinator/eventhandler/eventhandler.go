package eventhandler

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/alibaba/kubedl/pkg/features"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/core"
)

func NewEnqueueForObject(co core.Coordinator) handler.EventHandler {
	return &EnqueueForObject{Coordinator: co}
}

var _ handler.EventHandler = &EnqueueForObject{}

type EnqueueForObject struct {
	Coordinator core.Coordinator
}

// Create implements EventHandler
func (e *EnqueueForObject) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}

	if e.shouldSetOwnerQueueAndBreak(evt.Object.GetUID()) {
		e.Coordinator.SetQueueUnitOwner(evt.Object.GetUID(), q)
		return
	}

	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

// Update implements EventHandler
func (e *EnqueueForObject) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectNew != nil {

		if e.shouldSetOwnerQueueAndBreak(evt.ObjectNew.GetUID()) {
			e.Coordinator.SetQueueUnitOwner(evt.ObjectNew.GetUID(), q)
			return
		}

		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      evt.ObjectNew.GetName(),
			Namespace: evt.ObjectNew.GetNamespace(),
		}})
	}
}

// Delete implements EventHandler
func (e *EnqueueForObject) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		klog.Errorf("CreateEvent received with no metadata, event: %v", evt)
		return
	}

	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) && e.Coordinator != nil {
		// for delete events, just dequeue it from coordinator.
		e.Coordinator.Dequeue(evt.Object.GetUID())
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

// Generic implements EventHandler
func (e *EnqueueForObject) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		klog.Errorf("CreateEvent received with no metadata, event: %v", evt)
		return
	}
	if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) && e.Coordinator != nil {
		// for generic handler, it is called only when unknown type event arrives,
		// hence we set queue unit owner work-queue when it is queuing, otherwise
		// it will be dequeued.
		if e.Coordinator.IsQueuing(evt.Object.GetUID()) {
			// set queue unit owner work-queue and Coordinator will release it later.
			e.Coordinator.SetQueueUnitOwner(evt.Object.GetUID(), q)
		} else {
			e.Coordinator.Dequeue(evt.Object.GetUID())
		}
	}

	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

// shouldSetOwnerQueueAndBreak sets queue unit owner work-queue and Coordinator will release it later,
// if the arrived events is not queuing, it must be dequeued before and should add to real work-queue
// directly.
func (e *EnqueueForObject) shouldSetOwnerQueueAndBreak(uid types.UID) bool {
	return features.KubeDLFeatureGates.Enabled(features.JobCoordinator) && e.Coordinator != nil && e.Coordinator.IsQueuing(uid)
}
