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
	"k8s.io/utils/strings/slices"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
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
//
//	job interface{},
//	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec,
//	jobStatus apiv1.JobStatus,
//	runPolicy *apiv1.RunPolicy) error
type JobController struct {
	Controller apiv1.ControllerInterface

	Config options.JobControllerConfiguration
	// PodControl knows how to add or delete pods created as an interface to allow testing.
	PodControl controller.PodControlInterface

	// ServiceControl knows how to add or delete services created as an interface to allow testing.
	ServiceControl ServiceControlInterface

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
	patcher func(oldObj, newObj client.Object) error

	// Client talks to api-server and knows how to perform CRUD operations on Kubernetes objects.
	Client client.Client

	// Scheme defines methods for serializing and deserializing API objects
	Scheme *runtime.Scheme

	// APIReader knows how to read and list Kubernetes objects bypass cache to avoid retrieving
	// stale status for the reason of etcd slow-watch.
	APIReader client.Reader
}

func NewJobController(
	mgr controllerruntime.Manager,
	controllerImpl apiv1.ControllerInterface,
	config options.JobControllerConfiguration,
	recorder record.EventRecorder,
	metrics *metrics.JobMetrics,
	scheme *runtime.Scheme,
) JobController {
	jc := JobController{
		Controller:         controllerImpl,
		Config:             config,
		Expectations:       controller.NewControllerExpectations(),
		BackoffStatesQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Recorder:           recorder,
		Metrics:            metrics,
		Client:             mgr.GetClient(),
		APIReader:          mgr.GetAPIReader(),
		Scheme:             scheme,
	}
	jc.patcher = func(oldObj, newObj client.Object) error {
		// deepcopy new object avoid of in-memory modifications being override by in-cluster object.
		newPatchObj := newObj.DeepCopyObject()
		return mgr.GetClient().Patch(context.Background(), newPatchObj.(client.Object), client.MergeFrom(oldObj))
	}
	jc.PodControl = NewPodControl(jc.Client, recorder)
	jc.ServiceControl = NewServiceControl(jc.Client, recorder)
	return jc
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

func (jc *JobController) checkCache(metaObject metav1.Object, cacheBackendSpec *cachev1alpha1.CacheBackendSpec) error {
	cacheBackend := &cachev1alpha1.CacheBackend{}
	cacheBackendName := cacheBackendSpec.CacheBackendName
	cacheBackendNameSpace := metaObject.GetNamespace()
	err := jc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: cacheBackendNameSpace,
		Name:      cacheBackendName,
	}, cacheBackend)

	if err == nil {
		log.Infof("CacheBackend %s has been created", cacheBackendName)
		if !slices.Contains(cacheBackend.Status.UsedBy, metaObject.GetName()) {
			cacheCpy := cacheBackend.DeepCopy()
			cacheCpy.Status.UsedBy = append(cacheBackend.Status.UsedBy, metaObject.GetName())
			err = jc.Client.Status().Update(context.Background(), cacheCpy)
			if err != nil {
				log.Errorf("Update UsedBy status failed")
				return err
			}
		}
	} else {
		if k8serrors.IsNotFound(err) {
			// If haven't created yet
			if cacheBackendSpec.Dataset == nil || cacheBackendSpec.CacheEngine == nil {
				log.Errorf("Information on creating CacheBackend is missing. The creation failed")
				return err
			}
			log.Infof("CacheBackend is not exist, start to create %s", cacheBackendName)
			err = jc.createCache(cacheBackendName, cacheBackendNameSpace, cacheBackendSpec)
			if err != nil {
				log.Errorf("Failed to create CacheBackend %s", cacheBackendName)
				return err
			}
			log.Infof("CacheBackend %s created", cacheBackendName)
		} else {
			log.Errorf("Failed to get CacheBackend %s", cacheBackendName)
			return err
		}
	}

	return nil
}

func (jc *JobController) createCache(name, namespace string, cacheBackendSpec *cachev1alpha1.CacheBackendSpec) error {
	cacheBackend := &cachev1alpha1.CacheBackend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *cacheBackendSpec,
		Status: cachev1alpha1.CacheBackendStatus{
			CacheStatus: cachev1alpha1.CacheCreating,
			UsedBy:      []string{name},
		},
	}

	err := jc.Client.Create(context.Background(), cacheBackend)
	return err
}

func (jc JobController) addCachePathToContainer(metaObject metav1.Object, cacheBackend *cachev1alpha1.CacheBackendSpec,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {

	cacheObj := &cachev1alpha1.CacheBackend{}
	err := jc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: metaObject.GetNamespace(),
		Name:      cacheBackend.CacheBackendName,
	}, cacheObj)

	if err != nil {
		log.Errorf("Failed to get cacheBackend instance %s when inject cache to cantainers", cacheBackend.CacheBackendName)
	}

	// Check whether the PVC has been created
	pvc := &v1.PersistentVolumeClaim{}
	//pvcName := cachectrl.GetCacheName(metaObject)
	pvcName := cacheBackend.CacheBackendName
	err = jc.Client.Get(context.Background(), types.NamespacedName{
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
			jc.addCacheVolumeToPodSpec(pvcName, cacheBackend.MountPath, &spec.Template)
		}

	} else {
		if k8serrors.IsNotFound(err) {
			log.Errorf("Cannot find pvc %s, waiting to be created", pvcName)
		} else {
			log.Errorf("Failed to get pvc %s", pvcName)
		}
	}
	return err
}

func (jc JobController) addCacheVolumeToPodSpec(pvcName, mountPath string, pod *v1.PodTemplateSpec) {
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
				Name: "cachevolume", MountPath: mountPath,
			})
	}
}

func (jc JobController) updateCacheUsedStatus(metaObject metav1.Object, cacheBackendSpec *cachev1alpha1.CacheBackendSpec) (err error) {
	cacheBackend := &cachev1alpha1.CacheBackend{}
	cacheBackendName := cacheBackendSpec.CacheBackendName
	cacheBackendNameSpace := metaObject.GetNamespace()
	err = jc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: cacheBackendNameSpace,
		Name:      cacheBackendName,
	}, cacheBackend)

	if err == nil {
		//for i, jobName := range cacheBackend.Status.UsedBy {
		//	if jobName == metaObject.GetName() {
		//		cacheBackend.Status.UsedBy = append(cacheBackend.Status.UsedBy[:i], cacheBackend.Status.UsedBy[i+1:]...)
		//		break
		//	}
		//}

		i := slices.Index(cacheBackend.Status.UsedBy, metaObject.GetName())
		if i != -1 {
			cacheBackend.Status.UsedBy = append(cacheBackend.Status.UsedBy[:i], cacheBackend.Status.UsedBy[i+1:]...)
			log.Infof("Update CacheBackend %s used status", cacheBackendName)
			err = jc.Client.Status().Update(context.Background(), cacheBackend)
			if err != nil {
				log.Errorf("Update CacheBackend %s used status", cacheBackendName)
				return err
			}
		}

	} else {
		if k8serrors.IsNotFound(err) {
			log.Errorf("Cannot find CacheBackend %s to update used status", cacheBackendName)
		} else {
			log.Errorf("Failed to get CacheBackend %s to update used status", cacheBackendName)
		}
	}
	return
}
