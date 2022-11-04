package helper

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/resource_utils"
)

func ToQueueUnit(object client.Object, replicas map[v1.ReplicaType]*v1.ReplicaSpec, status *v1.JobStatus,
	schedPolicy *v1.SchedulingPolicy) *jobcoordinator.QueueUnit {
	qu := &jobcoordinator.QueueUnit{
		Object: object,
		Status: status,
		Specs:  replicas,
	}
	qu.Resources, qu.SpotResources = resource_utils.JobResourceRequests(replicas)
	qu.SchedPolicy = schedPolicy
	if schedPolicy != nil && schedPolicy.Priority != nil {
		qu.Priority = pointer.Int32Ptr(*schedPolicy.Priority)
	}
	return qu
}

func PopulatePriorityValue(clientset client.Client, qu *jobcoordinator.QueueUnit) {
	if qu.Priority != nil {
		return
	}
	if qu.Priority == nil && qu.SchedPolicy != nil && qu.SchedPolicy.PriorityClassName != "" {
		pc := schedulingv1.PriorityClass{}
		if err := clientset.Get(context.Background(), types.NamespacedName{
			Name: qu.SchedPolicy.PriorityClassName,
		}, &pc); err == nil {
			qu.Priority = &pc.Value
		}
	}
}

func QueueStateMarker(clientset client.Client, recorder record.EventRecorder) func(qu *jobcoordinator.QueueUnit, reason string) error {
	return func(qu *jobcoordinator.QueueUnit, reason string) error {
		old := qu.Object.DeepCopyObject()

		msg := fmt.Sprintf("Job %s is queuing and waiting for being scheduled.", qu.Key())
		if reason == util.JobDequeuedReason {
			msg = fmt.Sprintf("Job %s is being dequeued and waiting for reconciling.", qu.Key())
		}
		if err := util.UpdateJobConditions(qu.Status, v1.JobQueuing, reason, msg); err != nil {
			return err
		}

		recorder.Event(qu.Object, corev1.EventTypeNormal, reason, msg)

		return clientset.Status().Patch(context.Background(), qu.Object, client.MergeFrom(old.(client.Object)))
	}
}
