package core

import "k8s.io/apimachinery/pkg/util/sets"

type QueueSelector interface {
	Next(queues map[string]*queue) (string, *queue)
}

func NewRoundRobinSelector() QueueSelector {
	return &roundRobinSelector{lastSelectedIndex: -1}
}

var _ QueueSelector = &roundRobinSelector{}

// TODO(qiukai.cqk): replace default selector with weighted-round-robin and
// take total pending resource requests as its input weight.
type roundRobinSelector struct {
	queueNames        []string
	lastSelectedIndex int
}

func (wrr *roundRobinSelector) Next(queues map[string]*queue) (string, *queue) {
	if wrr.queuesChanged(queues) {
		wrr.appendNewQueues(queues)
	}

	// No queue available yet.
	if len(wrr.queueNames) == 0 {
		return "", nil
	}

	index := (wrr.lastSelectedIndex + 1) % len(wrr.queueNames)
	name := wrr.queueNames[index]
	q := queues[name]
	wrr.lastSelectedIndex++
	return name, q
}

func (wrr *roundRobinSelector) queuesChanged(queues map[string]*queue) bool {
	if len(wrr.queueNames) == 0 {
		return true
	}
	// Queue will not be released once created, hence length of queues changes indicates that
	// we should update queueNames slice.
	return len(wrr.queueNames) != len(queues)
}

// appendNewQueues append new incoming queues and provides stable order for round-robin selection.
func (wrr *roundRobinSelector) appendNewQueues(queues map[string]*queue) {
	curQueueSet := sets.NewString(wrr.queueNames...)

	for name := range queues {
		if !curQueueSet.Has(name) {
			wrr.queueNames = append(wrr.queueNames, name)
		}
	}
}
