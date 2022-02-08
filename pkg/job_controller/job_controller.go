package job_controller

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	cachectrl "github.com/alibaba/kubedl/controllers/cache"
	"github.com/alibaba/kubedl/pkg/gang_schedule"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

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

	Config options.JobControllerConfiguration
	// podControl knows how to add or delete pods created as an interface to allow testing.
	podControl controller.PodControlInterface

	// serviceControl knows how to add or delete services created as an interface to allow testing.
	serviceControl ServiceControlInterface

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

	// Scheme defines methods for serializing and deserializing API objects
	Scheme *runtime.Scheme
}

func NewJobController(
	cli client.Client,
	controllerImpl apiv1.ControllerInterface,
	config options.JobControllerConfiguration,
	recorder record.EventRecorder,
	metrics *metrics.JobMetrics,
	scheme *runtime.Scheme,
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
		Client:         cli,
		podControl:     NewPodControl(cli, recorder),
		serviceControl: NewServiceControl(cli, recorder),
		Scheme:         scheme,
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

// CreateGang create a new gang schedule process, ensure the relationship between job, managed objects and
// gang entity always maintained, so the consistency of gang scheduling never breaks.
func (jc *JobController) CreateGang(job metav1.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, schedPolicy *apiv1.SchedulingPolicy) (runtime.Object, error) {
	gangEntity, err := jc.GangScheduler.CreateGang(job, replicas, schedPolicy)
	if err != nil {
		log.Errorf("failed to create gang schedule entity, gang scheduler: %s, err: %v", jc.GangScheduler.PluginName(), err)
		return nil, err
	}
	log.Infof("gang schedule created, job name: %s ,scheduler name: %s", job.GetName(), jc.GangScheduler.PluginName())
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

func (jc *JobController) createCache(metaObject metav1.Object, cacheBackendSpec *cachev1alpha1.CacheBackendSpec,
	jobStatus *apiv1.JobStatus) error {
	cacheBackend := &cachev1alpha1.CacheBackend{}
	cacheBackendName := cachectrl.GetCacheName(metaObject)
	cacheBackendNameSpace := metaObject.GetNamespace()
	err := jc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: cacheBackendNameSpace,
		Name:      cacheBackendName,
	}, cacheBackend)

	if err == nil {
		// Already been created
		log.Infof("cache backend has been created")
		return nil
	} else {
		if k8serrors.IsNotFound(err) {
			log.Infof("cache backend is not exist, start to create %s", cacheBackendName)

			// If haven't created yet
			cacheBackend = &cachev1alpha1.CacheBackend{}
			cacheBackend.Name = cacheBackendName
			cacheBackend.Namespace = cacheBackendNameSpace
			cacheBackend.Spec = *cacheBackendSpec
			controllerRef := jc.GenOwnerReference(metaObject)
			cacheBackend.OwnerReferences = append(cacheBackend.OwnerReferences, *controllerRef)
			err = jc.Client.Create(context.Background(), cacheBackend)
			if err != nil {
				log.Errorf("failed to create cache backend %s", cacheBackend.Name)
				return err
			}

			// Update job status
			jobStatus.CacheBackendName = cacheBackendName

			// Update cache backend status
			cacheCopy := cacheBackend.DeepCopy()
			cacheCopy.Status.JobName = metaObject.GetName()
			cacheCopy.Status.CacheStatus = cachev1alpha1.CacheCreating
			err = jc.Client.Status().Update(context.Background(), cacheCopy)
			if err != nil {
				log.Error(err, "failed to update job name", "cacheBackend", cacheBackend.Name)
				return err
			}

		} else {
			log.Errorf("failed to get cache backend %s", cacheBackend.Name)
			return err
		}
	}
	log.Infof("cache backend %s created", cacheBackendName)
	return nil
}

func (jc JobController) addCachePathToContainer(metaObject metav1.Object, cacheBackend *cachev1alpha1.CacheBackendSpec,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {

	// Check whether the PVC has been created
	pvc := &v1.PersistentVolumeClaim{}
	pvcName := cachectrl.GetCacheName(metaObject)
	err := jc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: metaObject.GetNamespace(),
		Name:      pvcName,
	}, pvc)

	// pvc has been created, inject it to container
	if err == nil {
		for _, spec := range replicas {
			containerList := spec.Template.Spec.Containers
			for key, container := range containerList {
				exists := false
				for _, env := range container.Env {
					if env.Name == cachev1alpha1.KUBEDL_CACHE_NAME {
						exists = true
						break
					}
				}
				// append if not exists
				if !exists {
					containerList[key].Env = append(containerList[key].Env, v1.EnvVar{
						Name:  cachev1alpha1.KUBEDL_CACHE_NAME,
						Value: pvcName,
					})
				}
			}
			jc.addCacheVolumeToPodSpec(pvcName, cacheBackend, &spec.Template)
		}

	} else {
		if k8serrors.IsNotFound(err) {
			log.Errorf("cannot find pvc %s, waiting to be created", pvcName)
		} else {
			log.Errorf("fail to get pvc %s", pvcName)
		}
	}
	return err
}

func (jc JobController) addCacheVolumeToPodSpec(pvcName string, cacheBackend *cachev1alpha1.CacheBackendSpec, pod *v1.PodTemplateSpec) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		v1.Volume{
			Name: "cachevolume",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			}})

	for i, c := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(c.VolumeMounts,
			v1.VolumeMount{
				Name: "cachevolume", MountPath: cacheBackend.MountPath,
			})
	}
}
