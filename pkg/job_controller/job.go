package job_controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appv1 "github.com/alibaba/kubedl/apis/apps/v1alpha1"
	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	inference "github.com/alibaba/kubedl/apis/inference/v1alpha1"
	"github.com/alibaba/kubedl/apis/model/v1alpha1"
	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	model "github.com/alibaba/kubedl/controllers/model"
	"github.com/alibaba/kubedl/controllers/model/storage"
	"github.com/alibaba/kubedl/pkg/code_sync"
	"github.com/alibaba/kubedl/pkg/features"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	pkgruntime "github.com/alibaba/kubedl/pkg/util/runtime"
)

// Reasons for job events.
const (
	FailedDeleteJobReason     = "FailedDeleteJob"
	SuccessfulDeleteJobReason = "SuccessfulDeleteJob"
)

func (jc *JobController) deletePodsAndServices(runPolicy *apiv1.RunPolicy, job interface{}, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None.
	if *runPolicy.CleanPodPolicy == apiv1.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *runPolicy.CleanPodPolicy == apiv1.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
			continue
		}
		runtimeJob, ok := job.(runtime.Object)
		if !ok {
			return fmt.Errorf("%+v is not a runtime job", runtimeJob)
		}
		if err := jc.PodControl.DeletePod(pod.Namespace, pod.Name, runtimeJob); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := jc.ServiceControl.DeleteService(pod.Namespace, pod.Name, runtimeJob); err != nil {
			return err
		}
	}
	return nil
}

// ReconcileJobs checks and updates replicas for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (jc *JobController) ReconcileJobs(job client.Object, replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, jobStatus apiv1.JobStatus,
	runPolicy *apiv1.RunPolicy, modelVersion *v1alpha1.ModelVersionSpec, cacheBackendSpec *cachev1alpha1.CacheBackendSpec) (result reconcile.Result, err error) {

	jobName := job.GetName()
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return result, err
	}
	log.Infof("Reconciling for job %s", job.GetName())

	// if it's a scheduled task，create a cronJob of the same name
	if runPolicy != nil && runPolicy.CronPolicy != nil {
		if err := jc.ReconcileCron(job, job, runPolicy); err != nil {
			return result, err
		}
	}
	defer func() {
		// Add job key into backoff-states queue since it will be retried in
		// next round util reconciling succeeded.
		if result.Requeue || err != nil {
			jc.BackoffStatesQueue.AddRateLimited(jobKey)
			return
		}
		// Job exits reconciling process and do not have to be retried, just
		// forget it.
		jc.BackoffStatesQueue.Forget(jobKey)
	}()

	oldStatus := jobStatus.DeepCopy()

	err = code_sync.InjectCodeSyncInitContainers(job, replicas)
	if err != nil {
		log.Error(err, " Failed to inject code sync init container")
		return reconcile.Result{}, err
	}
	// TODO(SimonCqk): update job conditions failed ?

	if cacheBackendSpec != nil && jobStatus.CacheBackendName == "" && !commonutil.IsSucceeded(jobStatus) {
		// Check CacheBackend has been created or not
		err = jc.checkCache(job, cacheBackendSpec)

		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}

		// Next step is to get pvc and inject it to containers

		err = jc.addCachePathToContainer(job, cacheBackendSpec, replicas)
		if err != nil {
			log.Error(err, " Failed to inject pvc to containers")
			return reconcile.Result{Requeue: true}, err
		}

		jobStatus.CacheBackendName = cacheBackendSpec.CacheBackendName // update to api-server later
	}

	pods, err := jc.Controller.GetPodsForJob(job)
	if err != nil {
		log.Warnf("GetPodsForJob error %v", err)
		return result, err
	}

	services, err := jc.Controller.GetServicesForJob(job)
	if err != nil {
		log.Warnf("GetServicesForJob error %v", err)
		return result, err
	}

	// retrieve the previous number of retry
	previousRetry := jc.BackoffStatesQueue.NumRequeues(jobKey)

	activePods := k8sutil.FilterActivePods(pods)
	active := int32(len(activePods))
	failed := k8sutil.FilterPodCount(pods, v1.PodFailed)
	totalReplicas := k8sutil.GetTotalReplicas(replicas)
	prevReplicasFailedNum := k8sutil.GetTotalFailedReplicas(jobStatus.ReplicaStatuses)

	var failureMessage string
	jobExceedsLimit := false
	exceedsBackoffLimit := false
	pastBackoffLimit := false

	if runPolicy.BackoffLimit != nil {
		jobHasNewFailure := failed > prevReplicasFailedNum
		// new failures happen when status does not reflect the failures and active
		// is different than parallelism, otherwise the previous controller loop
		// failed updating status so even if we pick up failure it is not a new one
		exceedsBackoffLimit = jobHasNewFailure && (active != totalReplicas) &&
			(int32(previousRetry)+1 > *runPolicy.BackoffLimit)

		pastBackoffLimit, err = jc.pastBackoffLimit(jobName, runPolicy, replicas, pods)
		if err != nil {
			return result, err
		}
	}

	if exceedsBackoffLimit || pastBackoffLimit {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncJob
		jobExceedsLimit = true
		failureMessage = fmt.Sprintf("Job %s has failed because it has reached the specified backoff limit", jobName)
	} else if jc.pastActiveDeadline(runPolicy, jobStatus) {
		failureMessage = fmt.Sprintf("Job %s has failed because it was active longer than specified deadline", jobName)
		jobExceedsLimit = true
		now := metav1.Now()
		jobStatus.CompletionTime = &now
	}

	// If the Job is terminated, delete all pods and services.
	if commonutil.IsSucceeded(jobStatus) || commonutil.IsFailed(jobStatus) || jobExceedsLimit {
		if err = jc.deletePodsAndServices(runPolicy, job, pods); err != nil {
			return result, err
		}

		if result, err = jc.cleanupJob(runPolicy, jobStatus, job); err != nil {
			return result, err
		}

		if jc.Config.EnableGangScheduling {
			jc.Recorder.Event(job, v1.EventTypeNormal, "JobTerminated", "Job has been terminated. Deleting PodGroup")
			if err = jc.DeleteGang(job); err != nil {
				jc.Recorder.Eventf(job, v1.EventTypeWarning, "FailedDeletePodGroup", "Error deleting: %v", err)
				return result, err
			} else {
				jc.Recorder.Eventf(job, v1.EventTypeNormal, "SuccessfulDeletePodGroup", "Deleted PodGroup: %v", jobName)
			}
		}

		if jobExceedsLimit {
			jc.Recorder.Event(job, v1.EventTypeNormal, commonutil.JobFailedReason, failureMessage)
			if jobStatus.CompletionTime == nil {
				now := metav1.Now()
				jobStatus.CompletionTime = &now
			}
			err = commonutil.UpdateJobConditions(&jobStatus, apiv1.JobFailed, commonutil.JobFailedReason, failureMessage)
			if err != nil {
				log.Infof("Append job condition error: %v", err)
				return result, err
			}
		}

		// At this point the pods may have been deleted.
		// 1) If the job succeeded, we manually set the replica status.
		// 2) If any replicas are still active, set their status to succeeded.
		if commonutil.IsSucceeded(jobStatus) {
			for rtype := range jobStatus.ReplicaStatuses {
				jobStatus.ReplicaStatuses[rtype].Succeeded += jobStatus.ReplicaStatuses[rtype].Active
				jobStatus.ReplicaStatuses[rtype].Active = 0
			}

			// job finished, create the model version
			if modelVersion != nil {
				err = jc.createModelVersion(job, modelVersion, pods, &jobStatus)
			}
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
		}

		if cacheBackendSpec != nil {
			err = jc.updateCacheUsedStatus(job, cacheBackendSpec)
			if err != nil {
				return result, err
			}
		}

		if !reflect.DeepEqual(*oldStatus, jobStatus) {
			return result, jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus)
		}
		return result, nil
	}

	if jc.Config.EnableGangScheduling {
		log.Infof("gang schedule enabled, start to syncing for job %s", jobKey)
		if _, err = jc.CreateGang(job, replicas, runPolicy.SchedulingPolicy); err != nil {
			return result, err
		}
	}

	// Scaling will be triggered when following conditions satisfied:
	// 1. job has been in running state.
	// 2. elastic scaling feature gate has been enabled.
	// 3. generation has incremented, which represents the expected replicas changed.
	if commonutil.IsRunning(*oldStatus) && jc.Controller.EnableElasticScaling(job, runPolicy) {
		// Firstly check necessity of checkpoint, notify aimaster executing checkpoint if it is.
		// Once checkpoint triggered, scale out/in progress will be hold util it completed.
		done, err := jc.Controller.CheckpointIfNecessary(job, pods)
		if err != nil {
			log.Errorf("failed to checkpoint if necessary, err: %v", err)
			return result, err
		}
		// No in-progressing checkpoint and generation has incremented (scale out or scale in happened).
		if done && job.GetGeneration() > 1 {
			activeReplicasInNewGeneration := k8sutil.GetNumReplicasForLatestGeneration(pods, job.GetGeneration())

			if totalReplicas > activeReplicasInNewGeneration {
				err = jc.Controller.ScaleOut(job, replicas, pods, services)
			} else if totalReplicas < activeReplicasInNewGeneration {
				err = jc.Controller.ScaleIn(job, replicas, pods, services)
			}

			if err != nil {
				log.Errorf("failed to handle elastic scaling, err: %v", err)
				return result, err
			}
		}
	}

	// Save the current state of the replicas
	restart := false

	// add model path to container env
	addModelPathEnv(replicas, modelVersion)

	ctx := context.WithValue(context.Background(), apiv1.ContextHostNetworkPorts, make(map[string]int32))
	// Diff current active pods/services with replicas.
	for _, rtype := range jc.Controller.GetReconcileOrders() {
		spec, exist := replicas[rtype]
		if !exist {
			continue
		}

		// Hack for AIMaster and DO-NOT-REMOVE-THIS.
		if ContainsReplicaType(replicas, apiv1.JobReplicaTypeAIMaster) && rtype != apiv1.JobReplicaTypeAIMaster &&
			job.GetAnnotations()["aimaster"] != "ready" {
			log.Infof("aimaster has not ready and reconciling is freezed.")
			return reconcile.Result{}, nil
		}

		log.Infof("reconciles for replica type: %s", rtype)
		// If DAG scheduling has been enabled and current replica has upstream vertex(replica:phase),
		// wait util all upstream replicas ready then trigger next reconciling.
		if features.KubeDLFeatureGates.Enabled(features.DAGScheduling) && len(spec.DependOn) > 0 &&
			!jc.dagConditionsReady(job, replicas, pods, spec.DependOn) {
			continue
		}

		err = jc.ReconcilePods(ctx, job, &jobStatus, pods, rtype, spec, replicas, runPolicy, &restart)
		if err != nil {
			log.Warnf("ReconcilePods error %v", err)
			return result, err
		}

		if !jc.shouldCreateService(rtype) {
			continue
		}

		err = jc.ReconcileServices(ctx, job, services, rtype, spec)
		if err != nil {
			log.Warnf("ReconcileServices error %v", err)
			return result, err
		}
	}

	err = jc.Controller.UpdateJobStatus(job, replicas, &jobStatus, restart)
	if err != nil {
		log.Warnf("UpdateJobStatus error %v", err)
		return result, err
	}

	// Metering job status
	jc.Metrics.JobStatusMetrics(job, jobStatus)

	// Metering first pod launch delay when job state transit from created to running.
	if commonutil.IsCreated(*oldStatus) && commonutil.IsRunning(jobStatus) {
		jc.Metrics.FirstPodLaunchDelaySeconds(activePods, job, jobStatus)
	}

	// Metring all pods launch delay when latest pods are all active after reconciled, and previous
	// job status has not reached a all-active state, including the following cases:
	// 1. job created, successfully create all pods and becomes job running.
	// 2. job created, create some pods while some pods failed, finally becomes job running.
	// 3. job running then some pods failed, job step into restarting state, then pod recreated and
	//    finally return back to running state.
	//
	// case 3 should be discarded.
	if (k8sutil.GetTotalAvtiveReplicas(jobStatus.ReplicaStatuses) == totalReplicas) &&
		(k8sutil.GetTotalAvtiveReplicas(oldStatus.ReplicaStatuses) < totalReplicas) &&
		!commonutil.IsRestarting(*oldStatus) {
		jc.Metrics.AllPodsLaunchDelaySeconds(pods, job, jobStatus)
	}

	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, jobStatus) {
		if err = jc.Controller.UpdateJobStatusInApiServer(job, &jobStatus); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				result.Requeue = true
				return result, nil
			}
			return result, err
		}
	}
	return result, nil
}

func (jc *JobController) ReconcileCron(meta metav1.Object, runtimeObj runtime.Object, policy *apiv1.RunPolicy) error {
	cronJob := &appv1.Cron{}
	// check
	if policy == nil || policy.CronPolicy == nil {
		return nil
	}
	if err := jc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: meta.GetNamespace(),
		Name:      meta.GetName(),
	}, cronJob); err != nil {
		if errors.IsNotFound(err) {
			instance, err := jc.createCronInstance(policy, runtimeObj)
			if err != nil {
				return err
			}
			return jc.Client.Create(context.Background(), instance)
		}
		return err
	}

	newInstance, err := jc.createCronInstance(policy, runtimeObj)
	if err != nil {
		return err
	}
	if err := jc.updateCronJob(newInstance, cronJob); err != nil {
		return err
	}
	return nil
}

func (jc *JobController) updateCronJob(newInstance *appv1.Cron, oldCronJob *appv1.Cron) error {
	// If spec or ownerReference change, the cronJob needs to be updated
	dupCronJob := oldCronJob.DeepCopy()
	if !reflect.DeepEqual(dupCronJob.Spec, newInstance.Spec) {
		dupCronJob.Spec = newInstance.Spec
		return jc.Client.Patch(context.Background(), dupCronJob, client.MergeFrom(oldCronJob))
	}
	return nil
}

func (jc *JobController) createCronInstance(policy *apiv1.RunPolicy, runtimeObj runtime.Object) (*appv1.Cron, error) {
	tempPolicy := policy.DeepCopy()
	defer func() {
		policy.CronPolicy = tempPolicy.CronPolicy
	}()
	// Clear the CronPolicy under the cronjob created by simulation
	// If not cleared, the cronJob created here will continue to
	// create a cronJob, resulting in a dead cycle
	policy.CronPolicy = nil
	kind := runtimeObj.GetObjectKind().GroupVersionKind().Kind
	apiversion := runtimeObj.GetObjectKind().GroupVersionKind().GroupVersion().String()
	extCodec := pkgruntime.NewRawExtensionCodec(jc.Scheme)
	newMeta := runtimeObj.DeepCopyObject().(metav1.Object)
	metaName := newMeta.GetName()
	metaNs := newMeta.GetNamespace()
	// The new Cron only need Spec info, but Spec cannot get here,
	// it can only be converted into metaObj by force, then cleared
	// some attributes of meta. The overall framework will be refactored
	// and optimized in the future
	resetNonInheritableMetaFields(newMeta)
	rawObj, err := extCodec.EncodeRaw(newMeta.(runtime.Object))
	if err != nil {
		return nil, err
	}
	// create
	cronTemplateSpec := appv1.CronTemplateSpec{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiversion,
		},
		Workload: rawObj,
	}
	instance := &appv1.Cron{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metaName,
			Namespace: metaNs,
		},
		Spec: appv1.CronSpec{
			CronPolicy:   *tempPolicy.CronPolicy,
			CronTemplate: cronTemplateSpec,
		},
	}
	return instance, nil
}

// Extract the submitted job configuration to generate a cron
// job configuration, and set the parameters that cannot
// be assigned to empty.
func resetNonInheritableMetaFields(newMeta metav1.Object) {
	newMeta.SetName("")
	newMeta.SetNamespace("")
	newMeta.SetResourceVersion("")
	newMeta.SetUID("")
	newMeta.SetGeneration(0)
	newMeta.SetCreationTimestamp(metav1.Time{})
}

// addModelPathEnv add the model path into each container's env
// can be moved to webhook
func addModelPathEnv(replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, modelVersion *v1alpha1.ModelVersionSpec) {
	if modelVersion == nil {
		return
	}
	provider := storage.GetStorageProvider(modelVersion.Storage)
	for _, spec := range replicas {
		containerList := spec.Template.Spec.Containers
		for key, container := range containerList {
			exists := false
			for _, env := range container.Env {
				if env.Name == v1alpha1.KubeDLModelPath {
					exists = true
					break
				}
			}
			// append if not exists
			if !exists {
				containerList[key].Env = append(containerList[key].Env, v1.EnvVar{
					Name:  v1alpha1.KubeDLModelPath,
					Value: provider.GetModelMountPath(modelVersion.Storage),
				})
			}
		}

		// add the model volume to pod spec
		provider.AddModelVolumeToPodSpec(modelVersion.Storage, &spec.Template)
	}
}

func (jc *JobController) createModelVersion(job metav1.Object,
	modelVersion *v1alpha1.ModelVersionSpec, pods []*v1.Pod, jobStatus *apiv1.JobStatus) error {
	mv := &v1alpha1.ModelVersion{}
	// model version name, take the preceding 5 chars for versionId
	mvName := model.GetJobModelVersionName(job)
	err := jc.Client.Get(context.Background(), types.NamespacedName{
		Namespace: job.GetNamespace(),
		Name:      mvName,
	}, mv)

	if err == nil {
		// already exists
		return nil
	} else {
		if !errors.IsNotFound(err) {
			log.Errorf("failed to get model version %s", mv.Name)
			return err
		}
	}

	// create the new model version
	mv = &v1alpha1.ModelVersion{}
	mv.Namespace = job.GetNamespace()
	mv.Name = mvName
	mv.Spec = *modelVersion
	mv.Spec.CreatedBy = job.GetName()

	if mv.Spec.Storage != nil && mv.Spec.Storage.LocalStorage != nil {
		if mv.Spec.Storage.LocalStorage.NodeName != "" {
			mv.Spec.Storage.LocalStorage.NodeName = jc.Controller.GetNodeForModelOutput(pods)
		}
	}
	err = jc.Client.Create(context.Background(), mv)
	if err != nil {
		log.Errorf("failed to create model version %s", mv.Name)
		return err
	}

	jobStatus.ModelVersionName = mv.Name
	log.Infof("created model version %s", mv.Name)
	return nil
}

// pastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func (jc *JobController) pastActiveDeadline(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus) bool {
	if runPolicy.ActiveDeadlineSeconds == nil || jobStatus.StartTime == nil {
		return false
	}
	now := metav1.Now()
	start := jobStatus.StartTime.Time
	duration := now.Time.Sub(start)
	allowedDuration := time.Duration(*runPolicy.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

// pastBackoffLimit checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods with restartPolicy == OnFailure or Always
func (jc *JobController) pastBackoffLimit(jobName string, runPolicy *apiv1.RunPolicy,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, pods []*v1.Pod) (bool, error) {
	if runPolicy.BackoffLimit == nil {
		return false, nil
	}
	result := int32(0)
	for rtype, spec := range replicas {
		if spec.RestartPolicy != apiv1.RestartPolicyOnFailure && spec.RestartPolicy != apiv1.RestartPolicyAlways {
			log.Warnf("The restart policy of replica %v of the job %v is not OnFailure or Always. Not counted in backoff limit.", rtype, jobName)
			continue
		}
		// Convert ReplicaType to lower string.
		rt := strings.ToLower(string(rtype))
		pods, err := jc.FilterPodsForReplicaType(pods, rt)
		if err != nil {
			return false, err
		}
		for i := range pods {
			po := pods[i]
			if po.Status.Phase != v1.PodRunning {
				continue
			}
			for j := range po.Status.InitContainerStatuses {
				stat := po.Status.InitContainerStatuses[j]
				result += stat.RestartCount
			}
			for j := range po.Status.ContainerStatuses {
				stat := po.Status.ContainerStatuses[j]
				result += stat.RestartCount
			}
		}
	}

	if *runPolicy.BackoffLimit == 0 {
		return result > 0, nil
	}
	return result >= *runPolicy.BackoffLimit, nil
}

func (jc *JobController) cleanupJob(runPolicy *apiv1.RunPolicy, jobStatus apiv1.JobStatus, job client.Object) (reconcile.Result, error) {
	currentTime := time.Now()
	metaObject, _ := job.(metav1.Object)
	res := reconcile.Result{}
	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		return res, nil
	}
	if jobStatus.CompletionTime == nil {
		return res, fmt.Errorf("cleanup Job %s, but job has CompletionTime not set", metaObject.GetName())
	}
	duration := time.Second * time.Duration(*ttl)
	deleteTime := jobStatus.CompletionTime.Add(duration)
	if currentTime.After(deleteTime) {
		err := jc.Client.Delete(context.Background(), job)
		if err != nil {
			commonutil.LoggerForJob(metaObject).Warnf("Cleanup Job error: %v.", err)
			return res, err
		}
		return res, nil
	}
	res.Requeue = true
	res.RequeueAfter = deleteTime.Sub(currentTime)
	return res, nil
}

func (jc *JobController) shouldCreateService(rtype apiv1.ReplicaType) bool {
	if (jc.Controller.GetAPIGroupVersionKind().Kind == training.PyTorchJobKind && rtype != training.PyTorchReplicaTypeMaster) ||
		(jc.Controller.GetAPIGroupVersionKind().Kind == training.MPIJobKind) ||
		(jc.Controller.GetAPIGroupVersionKind().Kind == training.ElasticDLJobKind) ||
		(jc.Controller.GetAPIGroupVersionKind().Kind == inference.ElasticBatchJobKind) {
		return false
	}

	return true
}
