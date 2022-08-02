package recorder

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
)

func NewFlowControlledRecorder(recorder record.EventRecorder, qps int32) record.EventRecorder {
	if qps <= 0 {
		qps = 10
	}
	return &flowControlledRecorder{
		qps:            qps,
		recorder:       recorder,
		inflightEvents: make(map[types.UID]inflight),
		notify:         *sync.NewCond(&sync.Mutex{}),
	}
}

var _ record.EventRecorder = &flowControlledRecorder{}

type flowControlledRecorder struct {
	qps            int32
	recorder       record.EventRecorder
	inflightEvents map[types.UID]inflight
	notify         sync.Cond
	once           sync.Once
}

func (f *flowControlledRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	f.once.Do(func() {
		go wait.UntilWithContext(context.Background(), f.emitOne, time.Second/time.Duration(f.qps))
	})
	metaObj, ok := object.(metav1.Object)
	if !ok {
		f.recorder.Event(object, eventtype, reason, message)
		return
	}

	f.notify.L.Lock()
	defer f.notify.L.Unlock()

	// toNotify implies that previous pending events had all cleared out, and
	// emitOne() routine wait for being notified to consume events.
	toNotify := len(f.inflightEvents) == 0

	f.inflightEvents[metaObj.GetUID()] = inflight{
		obj:       object,
		eventType: eventtype,
		reason:    reason,
		msg:       message,
	}

	if toNotify {
		f.notify.Signal()
	}
}

func (f *flowControlledRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	f.Event(object, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (f *flowControlledRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	f.recorder.AnnotatedEventf(object, annotations, eventtype, reason, messageFmt, args...)
}

// emitOne randomly choose one pending event and emit it through real event recorder.
func (f *flowControlledRecorder) emitOne(_ context.Context) {
	if len(f.inflightEvents) == 0 {
		f.notify.Wait()
	}

	for uid, evt := range f.inflightEvents {
		f.recorder.Event(evt.obj, evt.eventType, evt.reason, evt.msg)
		f.notify.L.Lock()
		delete(f.inflightEvents, uid)
		if len(f.inflightEvents) == 0 {
			f.notify.Wait()
		}
		f.notify.L.Unlock()
		return
	}
}

type inflight struct {
	obj       runtime.Object
	eventType string
	reason    string
	msg       string
}
