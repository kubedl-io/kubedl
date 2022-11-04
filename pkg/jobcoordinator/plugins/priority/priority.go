package priority

import (
	"context"

	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/pkg/jobcoordinator"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/helper"
)

const (
	Name = "Priority"
)

func New(c client.Client, _ record.EventRecorder) jobcoordinator.Plugin {
	return &priority{clientSet: c}
}

var _ jobcoordinator.ScorePlugin = &priority{}

type priority struct {
	clientSet client.Client
}

// Score implements score interface and take priority of queue-unit as its score,
// which helps to determine the dequeue order.
func (p *priority) Score(_ context.Context, qu *jobcoordinator.QueueUnit) (int64, jobcoordinator.Status) {
	if qu.Priority == nil && qu.SchedPolicy != nil {
		if qu.SchedPolicy.Priority != nil {
			v := *qu.SchedPolicy.Priority
			qu.Priority = &v
		} else {
			helper.PopulatePriorityValue(p.clientSet, qu)
		}
	}

	value := int32(0)
	if qu.Priority != nil {
		value = *qu.Priority
	}

	klog.V(2).Infof("priority of queue unit %s is %v", qu.Key(), value)
	return int64(value), jobcoordinator.NewStatus(jobcoordinator.Success)
}

func (p *priority) Name() string {
	return Name
}
