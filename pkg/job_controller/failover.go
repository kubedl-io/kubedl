package job_controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	kruisev1alpha1 "github.com/openkruise/kruise/apis/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/concurrent"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	patchutil "github.com/alibaba/kubedl/pkg/util/patch"
	trainutil "github.com/alibaba/kubedl/pkg/util/train"
)

const (
	AnnotationAIMasterEnableErrorMonitoring = "pai.ai/enable-error-monitoring"
	AnnotationLastFailoverTimestamp         = v1.KubeDLPrefix + "/last-failover-timestamp"
	AnnotationImmediatelyStartWorker        = v1.KubeDLPrefix + "/immediately-start-worker"
	AnnotationImmediatelyRestartPod         = v1.KubeDLPrefix + "/immediately-restart-pod"
)

type FailOverAction string

const (
	FailOverInPlaceRestart FailOverAction = "InPlaceRestart"
	FailOverRecreate       FailOverAction = "Recreate"
)

// ShouldPodFailOver judges is input pod can be fail-overed or not, only replica with ExitCode or
// ExitCodeAThenRestart policy has chance to recreate and reschedule.
func ShouldPodFailOver(rspec *v1.ReplicaSpec, pod *corev1.Pod, exitCode int32) bool {
	if pod.Status.Phase == corev1.PodFailed && exitCode == 0 {
		klog.Warning("pod status conflicts: pod failed with exit code 0")
	}

	if rspec.RestartPolicy != v1.RestartPolicyExitCode {
		return false
	}

	return trainutil.IsRetryableExitCode(exitCode) || trainutil.IsRetryablePodFailedReason(pod.Status.Reason)
}

func (jc *JobController) DoFailOver(job client.Object, jobStatus *v1.JobStatus, rtype v1.ReplicaType, podsToFailover []*corev1.Pod) error {
	// Recreate pods who can be failovered first, pods will be deleted and create a new one
	// in next reconciliation, then being assigned to some other node after schedule.
	if err := jc.DoFailOverByAction(job, podsToFailover, FailOverRecreate); err != nil {
		return err
	}

	jc.Recorder.Eventf(job, corev1.EventTypeNormal, "FailOverSucceeded", "succeed to failover %d pods of job %s",
		len(podsToFailover), job.GetName())

	patch := patchutil.NewMergePatch()
	patch.InsertAnnotation(AnnotationLastFailoverTimestamp, time.Now().Format(time.RFC3339))
	return jc.Client.Patch(context.Background(), job, patch)
}

// DoFailOverByAction triggers failover by specified action, there are two different potential mechanism
// to resume the suspended job:
//
// - Recreate: delete anomalous one and create a new one, schedule to another node.
// - InPlaceRestart: restart containers of anomalous pod in-place and restart the process on same node.
func (jc *JobController) DoFailOverByAction(job client.Object, pods []*corev1.Pod, action FailOverAction) (err error) {

	klog.Infof("failover is triggered for job %s/%s, action: %v", job.GetNamespace(), job.GetName(), action)

	switch action {
	case FailOverRecreate:
		err = jc.RecreatePods(job, pods)
	case FailOverInPlaceRestart:
		err = jc.RestartPods(job, pods)
	}

	return err
}

func (jc *JobController) RecreatePods(job client.Object, pods []*corev1.Pod) error {
	klog.Infof("job %s/%s need to recreate %v pods", job.GetNamespace(), job.GetName(), len(pods))

	// limit max recreate parallelism as 50.
	tickets := 50
	if len(pods) < tickets {
		tickets = len(pods)
	}

	sema := concurrent.NewSemaphore(tickets)
	for _, p := range pods {
		sema.Acquire()
		go func(pod *corev1.Pod) {
			defer sema.Release()

			klog.Infof("pod %s/%s need to be recreated", pod.Namespace, pod.Name)
			if err := jc.PodControl.DeletePod(pod.Namespace, pod.Name, job); err != nil {
				klog.Errorf("failed to recreate pod %s/%s, err: %v", pod.Namespace, pod.Name, err)
			}
		}(p)
	}
	sema.Wait()
	return nil
}

func (jc *JobController) RestartPods(job client.Object, pods []*corev1.Pod) error {
	tickets := 50
	if len(pods) < tickets {
		tickets = len(pods)
	}

	total := len(pods)
	sema := concurrent.NewSemaphore(tickets)

	for _, p := range pods {
		if !k8sutil.IsPodActive(p) {
			klog.Infof("pod %s/%s is not active and skip inplace restart", p.Namespace, p.Name)
			continue
		}

		sema.Acquire()

		go func(pod *corev1.Pod) {
			defer sema.Release()

			if completed, err := jc.RestartPod(job, pod); err != nil {
				klog.Errorf("failed to restart pod, pod: %s, err: %v", pod.Name, err)
			} else {
				if completed {
					total--
				}
			}
		}(p)
	}
	// block until all semaphere is released.
	sema.Wait()
	return nil
}

func (jc *JobController) RestartPod(job client.Object, pod *corev1.Pod) (completed bool, err error) {
	podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	crr := kruisev1alpha1.ContainerRecreateRequest{}
	expectedGeneration := strconv.FormatInt(job.GetGeneration(), 10)
	err = jc.Client.Get(context.Background(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &crr)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, jc.inplaceRecreatePodContainers(job, pod, expectedGeneration)
		}

		klog.Errorf("failed to get latest container-recreate-request for stale worker, pod: %s", podKey)
		return false, err
	}

	// crr created in previous round, clean it.
	if crr.Labels[v1.LabelGeneration] != expectedGeneration {
		if err = jc.Client.Delete(context.Background(), &crr); err != nil {
			return false, err
		}
		return false, jc.inplaceRecreatePodContainers(job, pod, expectedGeneration)
	}

	if crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestFailed {
		jc.Recorder.Eventf(pod, corev1.EventTypeWarning, "FailedRestartContainer",
			"failed to restart containers of pod %s/%s, fallback to recreate pod", pod.Namespace, pod.Name)
		err = jc.Client.Delete(context.Background(), pod)
		return err == nil, err
	}

	recreateDone := crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestCompleted || crr.Status.Phase == kruisev1alpha1.ContainerRecreateRequestSucceeded
	if !recreateDone {
		klog.Infof("container recreate request has not completed yet, pod: %v", podKey)
		return false, nil
	}

	// Finalize container-recreate-request object once it completes, because elastic scaling is repeatable
	// and 'crr' request will be re-initiated.
	defer func() {
		_ = jc.Client.Delete(context.Background(), &crr)
	}()

	jc.Recorder.Eventf(pod, corev1.EventTypeNormal, "ContainerRecreateSucceed", "succeed to recreate containers in stale worker: %s", podKey)
	return true, nil
}

func (jc *JobController) inplaceRecreatePodContainers(job client.Object, pod *corev1.Pod, expectedGeneration string) error {
	crr := kruisev1alpha1.ContainerRecreateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Labels: map[string]string{
				v1.JobNameLabel:    job.GetName(),
				v1.LabelGeneration: expectedGeneration,
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
				*metav1.NewControllerRef(job, job.GetObjectKind().GroupVersionKind()),
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
	return jc.Client.Create(context.Background(), &crr)
}

func EnableErrorMonitoring(job client.Object) bool {
	return job.GetAnnotations()[AnnotationAIMasterEnableErrorMonitoring] == "true"
}

// podsToRestart defines the communication protocol between KubeDL and AIMaster,
// AIMaster pod analyze framework-level exceptions and asks KubeDL to restart candidate pods, KubeDL
// will reset this annotation after restart done, there is an example:
//
// {"worker":[1,3]} => indicates to restart worker-1 and worker-3.
type podsToRestart map[string][]int64

func getToBeRestartedPods(job client.Object) []string {
	if job.GetAnnotations()[AnnotationImmediatelyRestartPod] == "" {
		return nil
	}

	pods := make(podsToRestart)
	if err := json.Unmarshal([]byte(job.GetAnnotations()[AnnotationImmediatelyRestartPod]), &pods); err != nil {
		return nil
	}

	list := make([]string, 0, len(pods))
	for rtype, indexes := range pods {
		for _, index := range indexes {
			list = append(list, util.GenGeneralName(job.GetName(), rtype, strconv.FormatInt(index, 10)))
		}
	}
	return list
}
