package job_controller

import (
	"context"
	"strings"

	"github.com/alibaba/kubedl/pkg/gang_schedule"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
	log "github.com/sirupsen/logrus"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// JobControllerConfiguration contains configuration of operator.
type JobControllerConfiguration struct {
	// Enable gang scheduling by abstract GangScheduler.
	EnableGangScheduling bool

	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run.
	// Defaults to 1.
	MaxConcurrentReconciles int

	// ReconcilerSyncLoopPeriod is the amount of time the reconciler sync states loop
	// wait between two reconciler sync.
	// It is set to 15 sec by default.
	// TODO(cph): maybe we can let it grows by multiple in the future
	// and up to 5 minutes to reduce idle loop.
	// e.g. 15s, 30s, 60s, 120s...
	ReconcilerSyncLoopPeriod metav1.Duration

	// Name of global default gang scheduler.
	GangSchedulerName string

	// The container builder image name, Kaniko image
	ContainerImageBuilder string
}1

// JobController abstracts other operators to manage the lifecycle of Jobs.
// User need to first implement the ControllerInterface(objectA) and then initialize a JobController(objectB) struct with objectA
// as the parameter.
// And then call objectB.ReconcileJobs as mentioned below, the ReconcileJobs method is the entrypoint to trigger the
// reconcile logic of the job controller
//
// ReconcileJobs(
//		job interface{},
//		replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec,
//		jobStatus apiv1.JobStatus,
//		runPolicy *apiv1.RunPolicy) error
type JobController struct {
	Controller apiv1.ControllerInterface

	Config JobControllerConfiguration

	// Gang Scheduler is a abstract gang scheduling clientset.
	GangScheduler gang_schedule.GangScheduler

	// A TTLCache of pod/services creates/deletes each job expects to see
	// We use Job namespace/name + ReplicaType + pods/services as an expectation key,
	// For example, there is a TFJob with namespace "tf-operator" and name "tfjob-abc":
	// {
	//     "PS": {
	//         "Replicas": 2,
	//     },
	//     "Worker": {
	//         "Replicas": 4,
	//     }
	// }
	// We will create 4 expectations:
	// - "tf-operator/tfjob-abc/ps/services", expects 2 adds.
	// - "tf-operator/tfjob-abc/ps/pods", expects 2 adds.
	// - "tf-operator/tfjob-abc/worker/services", expects 4 adds.
	// - "tf-operator/tfjob-abc/worker/pods", expects 4 adds.
	Expectations controller.ControllerExpectationsInterface

	// BackoffStatesQueue is a rate limited queue and record backoff counts for
	// those reconciling-failed job instances, and it does not play a role of
	// build-in work queue in controller-runtime.
	BackoffStatesQueue workqueue.RateLimitingInterface

	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder

	// Metrics is a metrics exporter that export single numerical counter values.
	Metrics *metrics.JobMetrics

	// patcher creates a new patch differentiated from old and new object.
	patcher func(oldObj, newObj runtime.Object) error

	// Client talks to api-server
	Client client.Client
}

func NewJobController(
	cli client.Client,
	controllerImpl apiv1.ControllerInterface,
	config JobControllerConfiguration,
	recorder record.EventRecorder,
	metrics *metrics.JobMetrics,
) JobController {
	return JobController{
		Controller:         controllerImpl,
		Config:             config,
		Expectations:       controller.NewControllerExpectations(),
		BackoffStatesQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Recorder:           recorder,
		Metrics:            metrics,
		patcher: func(oldObj, newObj runtime.Object) error {
			// deepcopy new object avoid of in-memory modifications being override by in-cluster object.
			newPatchObj := newObj.DeepCopyObject()
			return cli.Patch(context.Background(), newPatchObj, client.MergeFrom(oldObj))
		},
		Client: cli,
	}
}

func (jc *JobController) GenOwnerReference(obj metav1.Object) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         jc.Controller.GetAPIGroupVersion().String(),
		Kind:               jc.Controller.GetAPIGroupVersionKind().Kind,
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func (jc *JobController) GenLabels(jobName string) map[string]string {
	labelGroupName := apiv1.GroupNameLabel
	labelJobName := apiv1.JobNameLabel
	groupName := jc.Controller.GetGroupNameLabelValue()
	return map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(jobName, "/", "-", -1),
	}
}

// CrateGang create a new gang schedule process, ensure the relationship between job, managed objects and
// gang entity always maintained, so the consistency of gang scheduling never breaks.
func (jc *JobController) CreateGang(job metav1.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) (runtime.Object, error) {
	gangEntity, err := jc.GangScheduler.GetGang(types.NamespacedName{
		Namespace: job.GetNamespace(),
		Name:      job.GetName(),
	})
	if err != nil {
		// Gang entity not found, create a new one.
		if k8serrors.IsNotFound(err) {
			gangEntity, err = jc.GangScheduler.CreateGang(job, replicas)
			if err != nil {
				log.Errorf("failed to create gang schedule entity, gang scheduler: %s, err: %v", jc.GangScheduler.Name(), err)
				return nil, err
			}
			log.Infof("gang schedule created, job name: %s ,scheduler name: %s", job.GetName(), jc.GangScheduler.Name())
		} else {
			return nil, err
		}
	}
	return gangEntity, nil
}

func (jc *JobController) DeleteGang(job metav1.Object) error {
	// Try deleting gang schedule entities firstly.
	err := jc.GangScheduler.DeleteGang(types.NamespacedName{
		Name:      job.GetName(),
		Namespace: job.GetNamespace(),
	})
	if err != nil {
		return err
	}
	log.Infof("Deleting GangSchedule %s", job.GetName())
	return nil
}

// resolveControllerRef returns the job referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching job
// of the correct Kind.
func (jc *JobController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) metav1.Object {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != jc.Controller.GetAPIGroupVersionKind().Kind {
		return nil
	}
	job, err := jc.Controller.GetJobFromInformerCache(namespace, controllerRef.Name)
	if err != nil {
		return nil
	}
	if job.GetUID() != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return job
}
