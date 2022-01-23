// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package job_controller

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	trainutil "github.com/alibaba/kubedl/pkg/util/train"
)

const (
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
)

// When a pod is created, enqueue the job that manages it and update its expectations.
func (jc *JobController) OnPodCreateFunc(e event.CreateEvent) bool {
	pod := e.Meta.(*v1.Pod)
	if pod.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		// jc.deletePod(pod)
		return false
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		job := jc.resolveControllerRef(pod.Namespace, controllerRef)
		logger := commonutil.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)
		if job == nil {
			if pod.Labels[apiv1.GroupNameLabel] == jc.Controller.GetGroupNameLabelValue() {
				logger.Infof("This pod's job does not exist, pod name: %s", pod.Name)
			}
			return false
		}

		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			logger.Infof("Failed to get the jobkey: %v", err)
			return false
		}

		rtype, ok := pod.Labels[apiv1.ReplicaTypeLabel]
		if !ok {
			logger.Infof("This pod maybe not created by %v, pod name: %s", jc.Controller.ControllerName(), pod.Name)
			return false
		}
		expectPodsKey := GenExpectationPodsKey(jobKey, rtype)
		jc.Expectations.CreationObserved(expectPodsKey)
		return true
	}
	return false
}

// When a pod is updated, figure out what job is managing it and wake it up. If the labels of
// the pod have changed we need to awaken both the old and new replica set, old and new must
// be *v1.Pod types.
func (jc *JobController) OnPodUpdateFunc(e event.UpdateEvent) bool {
	newPod := e.MetaNew.(*v1.Pod)
	oldPod := e.MetaOld.(*v1.Pod)
	if newPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return false
	}

	logger := commonutil.LoggerForPod(newPod, jc.Controller.GetAPIGroupVersionKind().Kind)
	newControllerRef := metav1.GetControllerOf(newPod)
	oldControllerRef := metav1.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(newControllerRef, oldControllerRef)

	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if job := jc.resolveControllerRef(oldPod.Namespace, oldControllerRef); job != nil {
			logger.Infof("pod controller ref updated: %v, %v", newPod, oldPod)
			return true
		}
	}

	// If it has a controller ref, that's all that matters.
	if newControllerRef != nil {
		job := jc.resolveControllerRef(newPod.Namespace, newControllerRef)
		if job == nil {
			return false
		}
		logger.Debugf("pod has a controller ref: %v, %v", newPod, oldPod)
		return true
	}
	return false
}

// When a pod is deleted, enqueue the job that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (jc *JobController) OnPodDeleteFunc(e event.DeleteEvent) bool {
	pod, ok := e.Meta.(*v1.Pod)
	logger := commonutil.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new job will not be woken up till the periodic resync.
	if e.DeleteStateUnknown {
		logger.Warnf("pod %s is in delete unknown state", pod.Name)
	}

	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return false
	}
	job := jc.resolveControllerRef(pod.Namespace, controllerRef)
	if job == nil {
		return false
	}
	jobKey, err := controller.KeyFunc(job)
	if err != nil {
		return false
	}
	rtype, ok := pod.Labels[apiv1.ReplicaTypeLabel]
	if !ok {
		logger.Infof("This pod maybe not created by %v", jc.Controller.ControllerName())
		return false
	}
	expectationPodsKey := GenExpectationPodsKey(jobKey, rtype)
	jc.Expectations.DeletionObserved(expectationPodsKey)
	return true
}

// FilterPodsForReplicaType returns pods belong to a replicaType.
func (jc *JobController) FilterPodsForReplicaType(pods []*v1.Pod, replicaType string) ([]*v1.Pod, error) {
	var result []*v1.Pod

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[apiv1.ReplicaTypeLabel] = replicaType

	for _, pod := range pods {
		selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
		if err != nil {
			return nil, err
		}
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		result = append(result, pod)
	}
	return result, nil
}

// GetPodSlices returns a slice, which element is the slice of pod.
func (jc *JobController) GetPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	podSlices := make([][]*v1.Pod, replicas)
	for _, pod := range pods {
		if _, ok := pod.Labels[apiv1.ReplicaIndexLabel]; !ok {
			logger.Warning("The pod do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(pod.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 {
			logger.Warningf("The label index is not expected: %d", index)
			continue
		} else if index >= replicas {
			// Pod index out of range, which indicates that it is a scale in
			// reconciliation and pod index>=replica will be deleted later, so
			// we'd increase capacity of pod slice to collect.
			newPodSlices := make([][]*v1.Pod, index+1)
			copy(newPodSlices, podSlices)
			podSlices = newPodSlices
		}
		podSlices[index] = append(podSlices[index], pod)
	}
	return podSlices
}

// ReconcilePods checks and updates pods for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods.
func (jc *JobController) ReconcilePods(
	ctx context.Context,
	job interface{},
	jobStatus *apiv1.JobStatus,
	pods []*v1.Pod,
	rtype apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec, restart *bool) error {

	metaObject, ok := job.(metav1.Object)
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	logger := commonutil.LoggerForReplica(metaObject, rt)
	// Get all pods for the type rt.
	pods, err := jc.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	var masterRole bool

	initializeReplicaStatuses(jobStatus, rtype)

	podSlices := jc.GetPodSlices(pods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rt, index)

			// check if this replica is the master role
			masterRole = jc.Controller.IsMasterRole(replicas, rtype, index)
			err = jc.createNewPod(ctx, job, rt, strconv.Itoa(index), spec, masterRole, replicas)
			if err != nil {
				// When controller tries to create a new pod but api-server returns a AlreadyExists error,
				// there may comes with two case:
				// 1. another controller watched this job instance and try to create pod concurrently.
				// 2. when this job was just created, there were some pods stuck at Terminating state
				//    and they belong to some job with same namespace/name.
				//
				// In the latter case, reconciling is interrupted and return a reconcile error, the underlying
				// work queue will requeue this request and try another round of reconciliation, however the
				// newly-arrived reconciliation just cancelled because no expectation satisfied, then no more
				// expected pods created. To fix this we generate a new expectation event when catch AlreadyExists
				// error.
				if errors.IsAlreadyExists(err) {
					jobKey, keyFuncErr := controller.KeyFunc(job)
					if keyFuncErr != nil {
						return err
					}

					expectationPodsKey := GenExpectationPodsKey(jobKey, rt)
					jc.Expectations.CreationObserved(expectationPodsKey)
					expectationServiceKey := GenExpectationServicesKey(jobKey, rt)
					jc.Expectations.CreationObserved(expectationServiceKey)

					logger.Infof("try create new pod %s but got a AlreadyExists error, generate a new expectation",
						commonutil.GenGeneralName(metaObject.GetName(), rt, strconv.Itoa(index)))
				}
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// Check if index is in valid range, otherwise we should scale in extra pods since
			// the expected replicas has been modified.
			if index < 0 || index >= numReplicas {
				if pod.DeletionTimestamp == nil {
					jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, "DeletePod",
						"pod %s/%s with index %v is out of expected replicas %v and should be deleted", pod.Namespace, pod.Name, index, numReplicas)
					if err = jc.podControl.DeletePod(pod.Namespace, pod.Name, runtimeObject); err != nil {
						return err
					}
				}
				continue
			}

			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == jc.Controller.GetDefaultContainerName() && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, exitedWithCodeReason, "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					break
				}
			}
			// Get and pass its container port by context if pod enables hostnetwork mode.
			if EnableHostNetwork(metaObject) {
				storeHostNetworkPortToContext(ctx, rt, strconv.Itoa(index),
					getContainerHostNetworkPort(pod, jc.Controller.GetDefaultContainerName(), jc.Controller.GetDefaultContainerPortName()))
			}

			// Check if the pod is retryable.
			if spec.RestartPolicy == apiv1.RestartPolicyExitCode {
				if pod.Status.Phase == v1.PodFailed &&
					(trainutil.IsRetryableExitCode(exitCode) || trainutil.IsRetryablePodFailedReason(pod.Status.Reason)) {
					logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if !ok {
						return fmt.Errorf("%+v is not a runtime job", runtimeObject)
					}
					if err = jc.podControl.DeletePod(pod.Namespace, pod.Name, runtimeObject); err != nil {
						return err
					}
					*restart = true
				}
			}

			updateJobReplicaStatuses(jobStatus, rtype, pod)
		}
	}
	return nil
}

// createNewPod creates a new pod for the given index and type.
func (jc *JobController) createNewPod(ctx context.Context, job interface{}, rt, index string, spec *apiv1.ReplicaSpec, masterRole bool,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {

	metaObject, ok := job.(metav1.Object)
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}

	logger := commonutil.LoggerForReplica(metaObject, rt)

	podTemplate := spec.Template.DeepCopy()

	// Set type and index for the worker.
	labels := jc.GenLabels(metaObject.GetName())
	labels[apiv1.ReplicaTypeLabel] = rt
	labels[apiv1.ReplicaIndexLabel] = index

	if masterRole {
		labels[apiv1.JobRoleLabel] = "master"
	}

	if EnableHostNetwork(metaObject) {
		commonutil.LoggerForReplica(metaObject, rt).Infof("pod enable host network, name: %s, masterRole: %v",
			metaObject.GetName(), masterRole)
		if err := jc.setupHostNetwork(ctx, podTemplate, rt, index); err != nil {
			return err
		}
	}

	podTemplate.Labels = commonutil.MergeMap(podTemplate.Labels, labels)

	if err := jc.Controller.SetClusterSpec(ctx, job, podTemplate, rt, index); err != nil {
		return err
	}

	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	if podTemplate.Spec.RestartPolicy != v1.RestartPolicy("") {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		jc.Recorder.Event(runtimeObject, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	setRestartPolicy(podTemplate, spec)

	// If gang-scheduling is enabled, try re-bind this pod with gang entity to maintain the gang relationship,
	// if it has existed, binding is a no-op.
	if jc.Config.EnableGangScheduling {
		entity, err := jc.GangScheduler.GetGang(types.NamespacedName{Name: metaObject.GetName(), Namespace: metaObject.GetNamespace()})
		if err != nil {
			return err
		}

		klog.V(5).Infof("gang scheduling enabled, gang scheduler name: %s, bind pod to gang: %s",
			jc.GangScheduler.PluginName(), metaObject.GetName())

		if err = jc.GangScheduler.BindPodToGang(metaObject, podTemplate, entity, rt); err != nil {
			return err
		}

		// 1) assign gang scheduler name if it's empty.
		// 2) override scheduler name if it differs from the selected gang implementation.
		podTemplate.Spec.SchedulerName = jc.GangScheduler.SchedulerName()
	}

	return jc.CreatePod(job, rt, index, podTemplate, masterRole)
}

// CreatePod creates a new common pod for the given index and type.
func (jc *JobController) CreatePod(job interface{}, rt, index string, podTemplate *v1.PodTemplateSpec, masterRole bool) error {
	metaObject, ok := job.(metav1.Object)
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}
	jobKey, err := KeyFunc(metaObject)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}
	expectationPodsKey := GenExpectationPodsKey(jobKey, rt)
	err = jc.Expectations.ExpectCreations(expectationPodsKey, 1)
	if err != nil {
		return err
	}

	// Set name for the template.
	podTemplate.Name = commonutil.GenGeneralName(metaObject.GetName(), rt, index)
	// Compatible with the naming convention of ElasticDL
	if jc.Controller.ControllerName() == "ElasticDLController" && masterRole {
		podTemplate.Name = "elasticdl-" + metaObject.GetName() + "-master"
	}

	controllerRef := jc.GenOwnerReference(metaObject)

	err = jc.podControl.CreatePodsWithControllerRef(metaObject.GetNamespace(), podTemplate, runtimeObject, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func (jc *JobController) setupHostNetwork(ctx context.Context, spec *v1.PodTemplateSpec, rtype, index string) error {
	port := int32(rand.IntnRange(jc.Config.HostNetworkPortRange.Base,
		jc.Config.HostNetworkPortRange.Base+jc.Config.HostNetworkPortRange.Size-1))
	// 1) enable pod hostnetwork mode.
	spec.Spec.HostNetwork = true
	// 2) [CRITICAL] setup dns policy with hostnet instead of ClusterFirst by default.
	spec.Spec.DNSPolicy = v1.DNSClusterFirstWithHostNet
	// 3) setup container port with a random port ranged [range_base, range_base+range_size).
	setupContainerHostNetworkPort(spec, jc.Controller.GetDefaultContainerName(), jc.Controller.GetDefaultContainerPortName(), port)
	// 4) record selected port by context keyed with replica-index.
	storeHostNetworkPortToContext(ctx, rtype, index, port)
	return nil
}

func setRestartPolicy(podTemplateSpec *v1.PodTemplateSpec, spec *apiv1.ReplicaSpec) {
	// This is necessary since restartPolicyExitCode is not supported in v1.PodTemplateSpec
	if spec.RestartPolicy == apiv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
	}
}

func (jc *JobController) AdoptAndClaimPods(job metav1.Object, podList *v1.PodList) ([]*v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: jc.GenLabels(job.GetName()),
	})
	if err != nil {
		return nil, err
	}
	pods := commonutil.ToPodPointerList(podList.Items)
	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := jc.Controller.GetJobFromAPIClient(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := controller.NewPodControllerRefManager(jc.podControl, job, selector, jc.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}
