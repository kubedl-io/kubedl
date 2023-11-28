package core

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/alibaba/kubedl/pkg/jobcoordinator"
)

func newQueue(name string) *queue {
	return &queue{
		name:    name,
		created: time.Now().Unix(),
		units:   make(map[types.UID]*jobcoordinator.QueueUnit),
	}
}

// queue is a simple queue-unit container implementation, unlike scheduling queue in
// kube-scheduler, the queue in job coordinator does not imply a FIFO model but holds
// all units together and coordinator select which one to pop out.
type queue struct {
	name    string
	created int64 // timestamp in unix
	units   map[types.UID]*jobcoordinator.QueueUnit
}

func (q *queue) add(qu *jobcoordinator.QueueUnit) {
	if q.exists(qu.Object.GetUID()) {
		q.update(qu)
		return
	}
	q.units[qu.Object.GetUID()] = qu
}

func (q *queue) remove(uid types.UID) {
	if !q.exists(uid) {
		return
	}
	delete(q.units, uid)
}

func (q *queue) update(qu *jobcoordinator.QueueUnit) {
	if !q.exists(qu.Object.GetUID()) {
		return
	}
	q.units[qu.Object.GetUID()] = qu
}

func (q *queue) get(uid types.UID) *jobcoordinator.QueueUnit {
	return q.units[uid]
}

func (q *queue) exists(uid types.UID) bool {
	_, ok := q.units[uid]
	return ok
}

func (q *queue) size() int {
	return len(q.units)
}

// snapshot snapshots a read-only view of current state of queue.
func (q *queue) snapshot() queueView {
	view := make(queueView, 0, len(q.units))
	for _, qu := range q.units {
		quc := *qu /* taking address is necessary */
		view = append(view, &quc)
	}
	return view
}

// iter returns an iterator for iterating each unit in queue view.
func (q *queue) iter() Iterator {
	return &queueIterator{q: q.snapshot()}
}

// Iterator defines how to iterate a units queue, and it is irreversible
// once pivot has incremented.
type Iterator interface {
	// HasNext indicates that is iterator at the ending position of queue.
	HasNext() bool
	// Next returns current unit pointed by pivot and increment pivot.
	Next() *jobcoordinator.QueueUnit
}

type queueView []*jobcoordinator.QueueUnit

// queueIterator is a queue iterator implementation and provides a thread-safe entry to view
// read-only elements in queue snapshot.
type queueIterator struct {
	mut   sync.RWMutex
	q     queueView
	pivot int
}

func (qi *queueIterator) HasNext() bool {
	qi.mut.RLock()
	defer qi.mut.RUnlock()
	return qi.pivot < len(qi.q)
}

func (qi *queueIterator) Next() *jobcoordinator.QueueUnit {
	if !qi.HasNext() {
		return nil
	}

	qi.mut.Lock()
	p := qi.pivot
	qi.pivot++
	qi.mut.Unlock()

	qu := qi.q[p]
	return qu
}
