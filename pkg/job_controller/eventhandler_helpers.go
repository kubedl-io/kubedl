package job_controller

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/alibaba/kubedl/pkg/features"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/core"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/helper"
	"github.com/alibaba/kubedl/pkg/metrics"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

type FieldExtractFunc func(object client.Object) (replicas map[v1.ReplicaType]*v1.ReplicaSpec, status *v1.JobStatus, schedPolicy *v1.SchedulingPolicy)

func OnOwnerCreateFunc(scheme *runtime.Scheme, fieldExtractFunc FieldExtractFunc, logger logr.Logger, coordinator core.Coordinator, metrics *metrics.JobMetrics) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		scheme.Default(e.Object)
		msg := fmt.Sprintf("Job %s/%s is created.", e.Object.GetObjectKind().GroupVersionKind().Kind, e.Object.GetName())
		replicas, status, schedPolicy := fieldExtractFunc(e.Object)

		if err := commonutil.UpdateJobConditions(status, v1.JobCreated, commonutil.JobCreatedReason, msg); err != nil {
			logger.Error(err, "append job condition error")
			return false
		}

		if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) && coordinator != nil &&
			(commonutil.NeedEnqueueToCoordinator(*status)) {
			logger.Info("job enqueued into coordinator and wait for being scheduled.", "namespace", e.Object.GetNamespace(), "name", e.Object.GetName())
			coordinator.EnqueueOrUpdate(helper.ToQueueUnit(e.Object, replicas, status, schedPolicy))
			return true
		}

		logger.Info(msg)
		metrics.CreatedInc()
		return true
	}
}

func OnOwnerUpdateFunc(scheme *runtime.Scheme, fieldExtractFunc FieldExtractFunc, logger logr.Logger, coordinator core.Coordinator) func(e event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		scheme.Default(e.ObjectOld)
		scheme.Default(e.ObjectNew)
		oldReplicas, _, oldSchedPolicy := fieldExtractFunc(e.ObjectOld)
		newReplicas, newStatus, newSchedPolicy := fieldExtractFunc(e.ObjectNew)

		if features.KubeDLFeatureGates.Enabled(features.JobCoordinator) && coordinator != nil {
			if commonutil.NeedEnqueueToCoordinator(*newStatus) {
				if !reflect.DeepEqual(oldReplicas, newReplicas) || !reflect.DeepEqual(oldSchedPolicy, newSchedPolicy) {
					coordinator.EnqueueOrUpdate(helper.ToQueueUnit(e.ObjectNew, newReplicas, newStatus, newSchedPolicy))
				}
			} else if coordinator.IsQueuing(e.ObjectNew.GetUID()) {
				coordinator.Dequeue(e.ObjectNew.GetUID())
			}
		}
		return true
	}
}

func OnOwnerDeleteFunc(jc JobController, fieldExtractFunc FieldExtractFunc, logger logr.Logger) func(e event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {
		replicas, _, _ := fieldExtractFunc(e.Object)
		logger.V(3).Info("job has been deleted", "namespace", e.Object.GetNamespace(), "name", e.Object.GetName())
		jc.DeleteExpectations(e.Object, replicas)
		return true
	}
}
