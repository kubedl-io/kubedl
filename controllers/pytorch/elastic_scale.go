package pytorch

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	kruisev1alpha1 "github.com/openkruise/kruise/apis/apps/v1alpha1"

	"github.com/alibaba/kubedl/apis"
	"github.com/alibaba/kubedl/pkg/util/concurrent"
	patchutil "github.com/alibaba/kubedl/pkg/util/patch"

	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
)

const (
	AnnotationCheckpointRequestedVersion = v1.KubeDLPrefix + "/ckpt-requested-version"
	AnnotationCheckpointCompletedVersion = v1.KubeDLPrefix + "/ckpt-completed-version"
	AnnotationReadyToStartWorker         = v1.KubeDLPrefix + "/ready-to-start-worker"
	AnnotationImmediatelyStartWorker     = v1.KubeDLPrefix + "/immediately-start-worker"
	AnnotationWorldSize                  = v1.KubeDLPrefix + "/world-size"
)

var nowFunc = metav1.Now

const (
	CheckpointStartReason    = "CheckpointStarted"
	CheckpointFinishedReason = "CheckpointSucceeded"
	CheckpointFailedReason   = "CheckpointFailed"
)

type ckptVersion struct {
	Version   int32       `json:"version"`
	Status    string      `json:"status"`
	Context   string      `json:"context"`
	Timestamp metav1.Time `json:"timestamp"`
}

const (
	checkpointInProgress = "InProgress"
	checkpointSucceeded  = "Succeeded"
	checkpointFailed     = "Failed"
)

func init() {
	apis.AddToSchemes = append(apis.AddToSchemes, kruisev1alpha1.AddToScheme)
}

func (r *PytorchJobReconciler) EnableElasticScaling(job client.Object, runPolicy *v1.RunPolicy) bool {
	return job.GetAnnotations()[v1.AnnotationEnableElasticTraining] == "true"
}

func (r *PytorchJobReconciler) ScaleOut(job client.Object, replicas map[v1.ReplicaType]*v1.ReplicaSpec, activePods []*corev1.Pod, activeServices []*corev1.Service) error {
	pytorchJob, ok := job.(*trainingv1alpha1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PytorchJob", job)
	}

	log.Info("start to scale out pytorch job", "key", pytorchJob.Namespace+"/"+pytorchJob.Name)

	_, err := r.scale(pytorchJob, replicas, activePods, activeServices)
	return err
}

func (r *PytorchJobReconciler) ScaleIn(job client.Object, replicas map[v1.ReplicaType]*v1.ReplicaSpec, activePods []*corev1.Pod, activeServices []*corev1.Service) error {
	pytorchJob, ok := job.(*trainingv1alpha1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PytorchJob", job)
	}

	replicasCounts := make(map[string]int64, len(replicas))
	for rtype, rspec := range replicas {
		replicasCounts[strings.ToLower(string(rtype))] = int64(*rspec.Replicas)
	}

	filteredPods := make([]*corev1.Pod, 0, len(activePods))
	filteredServices := make([]*corev1.Service, 0, len(activeServices))
	for _, p := range activePods {
		if !isIndexOutOfRange(p, replicasCounts[p.Labels[v1.ReplicaTypeLabel]]) {
			filteredPods = append(filteredPods, p)
		}
	}
	for _, svc := range activeServices {
		if !isIndexOutOfRange(svc, replicasCounts[svc.Labels[v1.ReplicaTypeLabel]]) {
			filteredServices = append(filteredServices, svc)
		}
	}

	log.Info("start to scale in pytorch job", "key", pytorchJob.Namespace+"/"+pytorchJob.Name)
	_, err := r.scale(pytorchJob, replicas, filteredPods, filteredServices)

	return err
}

// CheckpointIfNecessary triggers checkpoint when workers are going to be preempted or evicted, notify AIMaster
// to checkpoint and drain out victim pods after succeed.
// Checkpoint requests contains a `version` to distinguish from different progresses, and controller guarantees that
// 'checkpoint-version' <= 'job generation'. When preemption happens controller triggers a new round of checkpoint
// and take job generation as its version, and self-increase generation after checkpoint succeed.
func (r *PytorchJobReconciler) CheckpointIfNecessary(job client.Object, activePods []*corev1.Pod) (completed bool, err error) {
	pytorchJob, ok := job.(*trainingv1alpha1.PyTorchJob)
	if !ok {
		return false, fmt.Errorf("%+v is not a type of PytorchJob", job)
	}

	victims := filterVictimPods(activePods, pytorchJob.Generation)

	// start to notify aimaster executing checkpointing and wait for response.
	ckptReqVersion, err := getCheckpointVersion(pytorchJob.Annotations, AnnotationCheckpointRequestedVersion)
	if err != nil {
		return false, err
	}
	ckptCompletedVersion, err := getCheckpointVersion(pytorchJob.Annotations, AnnotationCheckpointCompletedVersion)
	if err != nil {
		return false, err
	}

	syncCheckpoint := ckptReqVersion == nil || (ckptCompletedVersion != nil && ckptReqVersion.Version == ckptCompletedVersion.Version)
	if syncCheckpoint {
		// SyncCheckpoint contains a 2-stage checkpoint transaction:
		// 1. complete.Version == request.Version indicates that job checkpoint has completed.
		// 2. request.Status (InProgress|Succeeded) tracks the whole progress, and job checkpoint
		//    is a sub-procedure of it. When complete.Version == request.Version satisfied in InProgress
		//    status, kubedl will clean up victim pods and increase job generation to windup whole progress
		//    and marks checkpoint as Succeeded. Newly observed victim pod events will be merged when status
		//    is InProgress, that is, a new round checkpoint will be triggered only when new victim pods
		//    observed in Succeeded status.

		// 1. Never execute checkpoint before then trigger first one.
		// 2. Previous checkpoint completed but new victim pod(s) observed, need to trigger
		// a new round of checkpoint.
		if ckptReqVersion == nil || ckptReqVersion.Status == checkpointSucceeded {
			if len(victims) == 0 {
				// no preemption happened, no need to checkpoint.
				log.Info("no preemption happened, no need to checkpoint")
				return true, nil
			}
			log.Info("need to trigger new checkpoint and wait for response",
				"key", pytorchJob.Namespace+"/"+pytorchJob.Name, "version", pytorchJob.Generation)
			r.recorder.Eventf(pytorchJob, corev1.EventTypeNormal, CheckpointStartReason,
				"start to checkpoint due to %d pod(s) is going to be evicted or drained out, version: %v", len(victims), pytorchJob.Generation)
			return false, r.triggerJobCheckpoint(pytorchJob)
		} else if ckptReqVersion.Status == checkpointInProgress {
			// A checkpoint has completed, cleanup victim pods, self-increase generation
			// and mark checkpoint as Succeeded to wind up.
			if err = r.cleanupVictimPods(pytorchJob, victims); err != nil {
				return false, err
			}
			if err = r.increaseGenerationAndMarkAsSucceeded(pytorchJob, ckptReqVersion); err != nil {
				return false, nil
			}
			log.Info("finish executing checkpoint for pytorch job", "job", pytorchJob.Name,
				"version", ckptCompletedVersion.Version, "completed timestamp", ckptCompletedVersion.Timestamp)
			r.recorder.Eventf(pytorchJob, corev1.EventTypeNormal, CheckpointFinishedReason,
				"finish executing checkpoint for pytorch job %s/%s, version: %v, context: %s",
				pytorchJob.Namespace, pytorchJob.Name, ckptCompletedVersion.Version, ckptCompletedVersion.Context)
			return true, nil
		}
	}

	log.Info("pytorch job checkpoint has not completed yet ...",
		"version", ckptReqVersion.Version, "start timestamp", ckptReqVersion.Timestamp)
	return false, nil
}

// scale handles elastic scaling driven by AIMaster, in general the steps of scale in/out workflow can be
// described as follows:
//
//  1. AIMaster updates expected replicas, then kubedl scales out new replicas or scales in extra replicas.
//  2. refresh master service to the latest generation, at this point no master endpoints will be selected,
//     and newly scaled pod will be injected an init container waiting for master service endpoint ready.
//  3. wait util AnnotationReadyToStartWorker is 'true', which represents worker checkpoint finished(controlled by AIMaster).
//  4. refresh stale master pod to the latest generation, after that master service will be available.
//  5. after 4), newly scaled pods will step into running state, then refresh stale worker pods one by one.
//  6. eventually no stale pods can be gathered, mark AnnotationReadyToStartWorker as false to end current round of scaling.
//
// The order of the above steps cannot be reversed.
func (r *PytorchJobReconciler) scale(pytorchJob *trainingv1alpha1.PyTorchJob, replicas map[v1.ReplicaType]*v1.ReplicaSpec, activePods []*corev1.Pod, activeServices []*corev1.Service) (finished bool, err error) {
	masterSvc := filterMasterService(activeServices)
	if masterSvc != nil {
		if err = r.refreshStaleService(masterSvc, pytorchJob.Generation); err != nil {
			return false, err
		}
	}

	if pytorchJob.Annotations[AnnotationReadyToStartWorker] != "true" && pytorchJob.Annotations[AnnotationImmediatelyStartWorker] != "true" {
		log.Info("pytorch has not ready to restart workers")
		return false, nil
	}
	// Update elastic scaling state as 'inflight' and it will be marked as 'done' when
	// process finishes.
	if pytorchJob.Annotations[v1.AnnotationElasticScaleState] != v1.ElasticScaleInflight {
		patch := patchutil.NewMergePatch()
		patch.InsertAnnotation(v1.AnnotationElasticScaleState, v1.ElasticScaleInflight)
		if err = r.Client.Patch(context.Background(), pytorchJob, patch); err != nil {
			return false, err
		}
	}

	totalReplicas := k8sutil.GetTotalExcludedReplicas(replicas, v1.JobReplicaTypeAIMaster)
	total, stalePods := k8sutil.FilterStalePodsByReplicaType(activePods, pytorchJob.Generation, strings.ToLower(string(v1.JobReplicaTypeAIMaster)))
	staleWorkers := stalePods[strings.ToLower(string(trainingv1alpha1.PyTorchReplicaTypeWorker))]
	staleMasters := stalePods[strings.ToLower(string(trainingv1alpha1.PyTorchReplicaTypeMaster))]

	masterCompleted := true
	for _, p := range staleMasters { // should only be one master.
		if completed, err := r.restartStalePod(pytorchJob, p, int64(totalReplicas), pytorchJob.Generation); err != nil {
			return false, err
		} else {
			masterCompleted = masterCompleted && completed
		}
	}
	if !masterCompleted {
		log.Info("refresh stale master has not completed yet", "key", pytorchJob.Namespace+"/"+pytorchJob.Name)
		return false, nil
	}
	total -= len(staleMasters)

	tickets := 100 // max semaphore tickets limited.
	if len(staleWorkers) < 100 {
		tickets = len(staleWorkers)
	}
	sema := concurrent.NewSemaphore(tickets)
	for _, p := range staleWorkers {
		sema.Acquire()

		go func(worker *corev1.Pod) {
			defer sema.Release()

			if completed, err := r.restartStalePod(pytorchJob, worker, int64(totalReplicas), pytorchJob.Generation); err != nil {
				klog.Errorf("failed to refresh stale worker, pod: %s, expected generation: %v, err: %v",
					worker.Name, pytorchJob.Generation, err)
			} else if completed {
				total--
			}
		}(p)
	}
	// block until all semaphere is released.
	sema.Wait()

	if len(stalePods) == 0 || total == 0 {
		log.Info("all pods are in latest generation, mark ready-to-start-worker as false")
		patch := patchutil.NewMergePatch()
		patch.InsertAnnotation(AnnotationReadyToStartWorker, "false")
		patch.InsertAnnotation(v1.AnnotationElasticScaleState, v1.ElasticScaleDone)
		if pytorchJob.Annotations[AnnotationImmediatelyStartWorker] == "true" {
			patch.InsertAnnotation(AnnotationImmediatelyStartWorker, "false")
		}
		if err = r.Client.Patch(context.Background(), pytorchJob, patch); err != nil {
			return false, err
		}
		r.recorder.Eventf(pytorchJob, corev1.EventTypeNormal, "ScaleSucceed",
			"pytorch job %s/%s elastic scaling successfully finished, total replicas: %v", pytorchJob.Namespace, pytorchJob.Name, totalReplicas)
		finished = true
	}
	return finished, nil
}

// restartStalePod refreshes stale pod to the latest generation, patch after-scaled-worldsize to
// pod annotation and triggers containers inplace restart, as for elastic scaling enabled pod,
// env WORLD-SIZE is referenced by annotation AnnotationWorldSize, new world-size value will be
// assigned to restarted pod.
func (r *PytorchJobReconciler) restartStalePod(job *trainingv1alpha1.PyTorchJob, pod *corev1.Pod, worldSize, generation int64) (completed bool, err error) {
	expectedWorldSize := strconv.FormatInt(worldSize, 10)
	expectedGeneration := strconv.FormatInt(generation, 10)
	podKey := pod.Namespace + "/" + pod.Name

	if job.Annotations[AnnotationImmediatelyStartWorker] == "true" && !k8sutil.IsPodActive(pod) {
		err = r.Client.Delete(context.Background(), pod)
		return err == nil, err
	}

	if pod.Labels[v1.LabelGeneration] == expectedGeneration {
		if ts := getLastRestartFinishTimestamp(pod, r.GetDefaultContainerName()); ts != nil {
			klog.Infof("pod %s/%s finished restart at %v", pod.Namespace, pod.Name, ts.String())
		}
		return true, nil
	}

	log.Info("refresh stale pod to latest generation", "pod", podKey, "generation", generation)

	completed, err = r.restartPodInKruiseProtocol(job, pod, expectedWorldSize, expectedGeneration)
	if !completed {
		return false, err
	}

	// Finally, incremental generation for current worker and mark refreshment done.
	patch := patchutil.NewStrategicPatch()
	patch.InsertLabel(v1.LabelGeneration, expectedGeneration)
	err = r.Client.Patch(context.Background(), pod, patch)
	if err != nil {
		return false, err
	}
	klog.Infof("succeed to refresh pod to generation: %v", generation)
	r.recorder.Eventf(pod, corev1.EventTypeNormal, "RefreshPodSucceed", "succeed to refresh pod to generation: %v", generation)
	return true, nil
}

func (r *PytorchJobReconciler) restartPodInKruiseProtocol(job *trainingv1alpha1.PyTorchJob, pod *corev1.Pod, expectedWorldSize, expectedGeneration string) (completed bool, err error) {
	podKey := pod.Namespace + "/" + pod.Name

	if curWorldSize, ok := pod.Annotations[AnnotationWorldSize]; !ok || curWorldSize != expectedWorldSize {
		log.Info("update latest world size of pytorch",
			"key", podKey, "current world size", curWorldSize, "target world size", expectedWorldSize)
		patch := patchutil.NewStrategicPatch()
		patch.InsertAnnotation(AnnotationWorldSize, expectedWorldSize)
		if err = r.Client.Patch(context.Background(), pod, patch); err != nil {
			log.Error(err, "failed to refresh world-size of stale worker", "pod", podKey, "world size", expectedWorldSize)
			return false, err
		}
		return false, nil
	}

	crr := kruisev1alpha1.ContainerRecreateRequest{}
	if err = r.Client.Get(context.Background(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &crr); err != nil {
		if errors.IsNotFound(err) {
			return false, r.recreatePodContainers(job, pod, expectedGeneration)
		}

		log.Error(err, "failed to get latest container-recreate-request for stale worker",
			"pod", podKey)
		return false, err
	}
	// crr created in previous round, clean it.
	if crr.Labels[v1.LabelGeneration] != expectedGeneration {
		if err = r.Client.Delete(context.Background(), &crr); err != nil {
			return false, err
		}
		return false, r.recreatePodContainers(job, pod, expectedGeneration)
	}
	if crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestFailed {
		r.recorder.Eventf(pod, corev1.EventTypeWarning, "FailedRestartContainer",
			"failed to restart containers of pod %s/%s, fallback to recreate pod", pod.Namespace, pod.Name)
		err = r.Client.Delete(context.Background(), pod)
		return err == nil, err
	}

	recreateDone := crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestCompleted || crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestSucceeded
	if !recreateDone {
		log.Info("container recreate request has not completed yet", "pod", podKey)
		return false, nil
	}

	// Finalize container-recreate-request object once it completes, because elastic scaling is repeatable
	// and 'crr' request will be re-initiated.
	defer func() {
		_ = r.Client.Delete(context.Background(), &crr)
	}()

	r.recorder.Eventf(pod, corev1.EventTypeNormal, "ContainerRecreateSucceed", "succeed to recreate containers in stale worker: %s", podKey)
	return true, nil
}

// refreshStaleService refreshes stale service, both in generation label and label selector
// embedded in spec, to the latest generation, only pods labeled with the latest generation
// will be selected by latest generated service.
func (r *PytorchJobReconciler) refreshStaleService(svc *corev1.Service, generation int64) error {
	expectedGeneration := strconv.FormatInt(generation, 10)
	if svc.Labels[v1.LabelGeneration] == expectedGeneration && svc.Spec.Selector[v1.LabelGeneration] == expectedGeneration {
		return nil
	}

	log.Info("refresh stale service to latest generation", "service", svc.Namespace+"/"+svc.Name, "generation", generation)

	svcCpy := svc.DeepCopy()
	svcCpy.Labels[v1.LabelGeneration] = expectedGeneration
	if svcCpy.Spec.Selector == nil {
		svcCpy.Spec.Selector = make(map[string]string)
	}
	svcCpy.Spec.Selector[v1.LabelGeneration] = expectedGeneration

	if err := r.Client.Patch(context.Background(), svcCpy, client.MergeFrom(svc)); err != nil {
		log.Error(err, "failed to refresh stale service", "service", svc.Namespace+"/"+svc.Name, "generation", generation)
		return err
	}
	r.recorder.Eventf(svc, corev1.EventTypeNormal, "RefreshServiceSucceed", "succeed to refresh service to generation: %v", generation)
	return nil
}

func (r *PytorchJobReconciler) recreatePodContainers(job *trainingv1alpha1.PyTorchJob, pod *corev1.Pod, generation string) error {
	crr := kruisev1alpha1.ContainerRecreateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Labels: map[string]string{
				v1.LabelGeneration: generation,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "Pod",
					Name:               pod.Name,
					UID:                pod.UID,
					Controller:         pointer.BoolPtr(false),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
				{
					APIVersion:         job.APIVersion,
					Kind:               job.Kind,
					Name:               job.Name,
					UID:                job.UID,
					Controller:         pointer.BoolPtr(false),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
			},
		},
		Spec: kruisev1alpha1.ContainerRecreateRequestSpec{
			PodName:  pod.Name,
			Strategy: &kruisev1alpha1.ContainerRecreateRequestStrategy{OrderedRecreate: false},
		},
	}

	for ci := range pod.Spec.Containers {
		container := &pod.Spec.Containers[ci]
		crr.Spec.Containers = append(crr.Spec.Containers, kruisev1alpha1.ContainerRecreateRequestContainer{Name: container.Name})
	}
	return r.Client.Create(context.Background(), &crr)
}

func (r *PytorchJobReconciler) triggerJobCheckpoint(pytorchJob *trainingv1alpha1.PyTorchJob) error {
	version := &ckptVersion{
		Version:   int32(pytorchJob.Generation),
		Context:   "pytorch job starts to request for checkpoint",
		Timestamp: nowFunc(),
		Status:    checkpointInProgress,
	}

	versionStr, err := json.Marshal(version)
	if err != nil {
		return err
	}
	patch := patchutil.NewMergePatch()
	patch.InsertAnnotation(AnnotationCheckpointRequestedVersion, string(versionStr))
	if err = r.Client.Patch(context.Background(), pytorchJob, patch); err != nil {
		return err
	}
	return nil
}

func (r *PytorchJobReconciler) cleanupVictimPods(job *trainingv1alpha1.PyTorchJob, victims []*corev1.Pod) error {
	sema := concurrent.NewSemaphore(10)
	errs := make(chan error, 10)
	for _, v := range victims {
		sema.Acquire()

		go func(p *corev1.Pod) {
			defer sema.Release()
			log.Info("pods to cleanup", "key", p.Namespace+"/"+p.Name)
			if err := r.ctrl.PodControl.DeletePod(p.Namespace, p.Name, job); err != nil {
				errs <- err
			}
		}(v)
	}
	sema.Wait()
	close(errs)

	if len(errs) == 0 {
		return nil
	}
	errStr := ""
	for err := range errs {
		errStr += fmt.Sprintf("%v\n", err)
	}
	return stderrors.New(errStr)
}

// increaseGenerationAndMarkSucceeded updates a trivial field in job spec to initiative update
// its generation.
func (r *PytorchJobReconciler) increaseGenerationAndMarkAsSucceeded(pytorchJob *trainingv1alpha1.PyTorchJob, ckptReqVersion *ckptVersion) error {
	if ckptReqVersion != nil {
		ckptReqVersion.Status = checkpointSucceeded
		anno, err := setCheckpointVersion(pytorchJob.Annotations, AnnotationCheckpointRequestedVersion, ckptReqVersion)
		if err != nil {
			return err
		}
		pytorchJob.Annotations = anno
	}

	pytorchJob.Annotations[AnnotationReadyToStartWorker] = "true"
	spec := pytorchJob.Spec.PyTorchReplicaSpecs[trainingv1alpha1.PyTorchReplicaTypeMaster]
	if spec == nil {
		spec = pytorchJob.Spec.PyTorchReplicaSpecs[trainingv1alpha1.PyTorchReplicaTypeWorker]
	}
	if spec.Template.Annotations == nil {
		spec.Template.Annotations = make(map[string]string)
	}
	spec.Template.Annotations["kubedl.io/generation-to-increase"] = strconv.FormatInt(pytorchJob.Generation+1, 10)
	if err := r.Client.Update(context.Background(), pytorchJob); err != nil {
		return err
	}
	return nil
}

func filterVictimPods(activePods []*corev1.Pod, latestGeneration int64) []*corev1.Pod {
	victims := make([]*corev1.Pod, 0, len(activePods))
	for _, p := range activePods {
		if k8sutil.IsVictimCandidatePod(p) {
			victims = append(victims, p)
		}
	}
	return victims
}

func filterMasterService(services []*corev1.Service) *corev1.Service {
	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[v1.ReplicaTypeLabel] = strings.ToLower(string(trainingv1alpha1.PyTorchReplicaTypeMaster))
	selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
	if err != nil {
		return nil
	}

	for _, service := range services {
		if selector.Matches(labels.Set(service.Labels)) {
			return service
		}
	}
	return nil
}

func AddMasterWaiterForWorker(podTemplate *corev1.PodTemplateSpec, param InitContainerParam) error {
	containers, err := renderMasterWaiterInitContainer(param)
	if err != nil {
		return err
	}
	podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers, containers...)
	return nil
}

func AddImageWarmupForWorker(podTemplate *corev1.PodTemplateSpec, mainContainerName string) {
	image := ""
	resources := corev1.ResourceRequirements{}
	for ci := range podTemplate.Spec.Containers {
		c := &podTemplate.Spec.Containers[ci]
		if c.Name == mainContainerName {
			image = c.Image
			c.Resources.DeepCopyInto(&resources)
			break
		}
	}
	if image == "" {
		return
	}
	initContainer := corev1.Container{
		Name:  "warmup",
		Image: image,
		Command: []string{
			"echo",
			"I do nothing but warmup image of main container",
		},
		Resources: resources,
	}
	if _, ok := resources.Requests[v1.ResourceNvidiaGPU]; ok {
		initContainer.Env = []corev1.EnvVar{
			{Name: "NVIDIA_VISIBLE_DEVICES", Value: ""},
			{Name: "NVIDIA_DRIVER_CAPABILITIES", Value: "all"},
		}
	}

	podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers, initContainer)
}

var initContainerTemplate = `
- name: master-waiter
  image: {{.InitContainerImage}}
  imagePullPolicy: IfNotPresent
  env:
  - name: MASTER_ADDR
    value: {{.MasterAddr}}
  command: ['sh', '-c', 'until ping -c1 $MASTER_ADDR >/dev/null 2>&1; do :; sleep 0.1; done;']`

type InitContainerParam struct {
	MasterAddr         string
	InitContainerImage string
}

func renderMasterWaiterInitContainer(param InitContainerParam) ([]corev1.Container, error) {
	var buf bytes.Buffer
	tpl, err := template.New("container").Parse(initContainerTemplate)
	if err != nil {
		return nil, err
	}
	if err := tpl.Execute(&buf, param); err != nil {
		return nil, err
	}

	var result []corev1.Container
	err = yaml.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func isIndexOutOfRange(obj metav1.Object, replicas int64) bool {
	index := obj.GetLabels()[v1.ReplicaIndexLabel]
	rtype := obj.GetLabels()[v1.ReplicaTypeLabel]
	if rtype == "" || index == "" {
		return false
	}
	indexNum, err := strconv.ParseInt(index, 10, 64)
	if err != nil {
		return false
	}
	return indexNum >= replicas
}

func getCheckpointVersion(anno map[string]string, key string) (*ckptVersion, error) {
	cvBytes := anno[key]
	if len(cvBytes) == 0 {
		return nil, nil
	}
	cv := ckptVersion{}
	if err := json.Unmarshal([]byte(cvBytes), &cv); err != nil {
		return nil, err
	}
	return &cv, nil
}

func setCheckpointVersion(anno map[string]string, key string, version *ckptVersion) (map[string]string, error) {
	if anno == nil {
		anno = make(map[string]string)
	}
	cvBytes, err := json.Marshal(version)
	if err != nil {
		return anno, nil
	}
	anno[key] = string(cvBytes)
	return anno, nil
}

func getLastRestartFinishTimestamp(p *corev1.Pod, containerName string) *metav1.Time {
	for ci := range p.Status.ContainerStatuses {
		cs := &p.Status.ContainerStatuses[ci]
		if cs.Name == containerName && cs.LastTerminationState.Terminated != nil {
			return &cs.LastTerminationState.Terminated.FinishedAt
		}
	}
	return nil
}
