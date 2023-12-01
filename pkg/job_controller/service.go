// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"

	log "github.com/sirupsen/logrus"

	"github.com/alibaba/kubedl/pkg/features"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
)

// When a service is created, enqueue the controller that manages it and update its expectations.
func (jc *JobController) OnServiceCreateFunc(e event.CreateEvent) bool {
	service := e.Object.(*v1.Service)
	if service.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new service shows up in a state that
		// is already pending deletion. Prevent the service from being a creation observation.
		// tc.deleteService(service)
		return false
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(service); controllerRef != nil {
		job := jc.resolveControllerRef(service.Namespace, controllerRef)
		if job == nil {
			return false
		}

		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			return false
		}

		rtype, ok := service.Labels[apiv1.ReplicaTypeLabel]
		if !ok {
			log.Infof("This service maybe not created by %v", jc.Controller.ControllerName())
			return false
		}
		expectationServicesKey := GenExpectationServicesKey(jobKey, rtype)
		jc.Expectations.CreationObserved(expectationServicesKey)
		return true
	}
	return false
}

// When a service is updated, figure out what job/s manage it and wake them up.
// If the labels of the service have changed we need to awaken both the old
// and new replica set. old and new must be *v1.Service types.
func (jc *JobController) OnServiceUpdateFunc(e event.UpdateEvent) bool {
	newService := e.ObjectNew.(*v1.Service)
	oldService := e.ObjectOld.(*v1.Service)
	if newService.ResourceVersion == oldService.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return false
	}

	newControllerRef := metav1.GetControllerOf(newService)
	oldControllerRef := metav1.GetControllerOf(oldService)
	controllerRefChanged := !reflect.DeepEqual(newControllerRef, oldControllerRef)

	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if job := jc.resolveControllerRef(oldService.Namespace, oldControllerRef); job != nil {
			return true
		}
	}

	// If it has a controller ref, that's all that matters.
	if newControllerRef != nil {
		job := jc.resolveControllerRef(newService.Namespace, newControllerRef)
		return job != nil
	}
	return false
}

// When a service is deleted, enqueue the job that manages the service and update its expectations.
// obj could be an *v1.Service, or a DeletionFinalStateUnknown marker item.
func (jc *JobController) OnServiceDeleteFunc(e event.DeleteEvent) bool {
	service := e.Object.(*v1.Service)

	// When a deletion is dropped, the relist will notice a service in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the service
	// changed labels the new job will not be woken up till the periodic resync.
	if e.DeleteStateUnknown {
		klog.Warningf("service %s is in delete unknown state", service.Name)
	}

	controllerRef := metav1.GetControllerOf(service)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return false
	}
	job := jc.resolveControllerRef(service.Namespace, controllerRef)
	if job == nil {
		return false
	}
	jobKey, err := controller.KeyFunc(job)
	if err != nil {
		return false
	}
	rtype, ok := service.Labels[apiv1.ReplicaTypeLabel]
	if !ok {
		klog.Infof("This service maybe not created by %v", jc.Controller.ControllerName())
		return false
	}
	expectationPodsKey := GenExpectationServicesKey(jobKey, rtype)
	jc.Expectations.DeletionObserved(expectationPodsKey)
	return true
}

// FilterServicesForReplicaType returns service belong to a replicaType.
func (jc *JobController) FilterServicesForReplicaType(services []*v1.Service, replicaType string) ([]*v1.Service, error) {
	var result []*v1.Service

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[apiv1.ReplicaTypeLabel] = replicaType

	for _, service := range services {
		selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
		if err != nil {
			return nil, err
		}
		if !selector.Matches(labels.Set(service.Labels)) {
			continue
		}
		result = append(result, service)
	}
	return result, nil
}

// GetServiceSlices returns a slice, which element is the slice of service.
// Assume the return object is serviceSlices, then serviceSlices[i] is an
// array of pointers to services corresponding to Services for replica i.
func (jc *JobController) GetServiceSlices(services []*v1.Service, replicas int, logger *log.Entry) [][]*v1.Service {
	serviceSlices := make([][]*v1.Service, replicas)
	for _, service := range services {
		if _, ok := service.Labels[apiv1.ReplicaIndexLabel]; !ok {
			logger.Warning("The service do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(service.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 {
			logger.Warningf("The label index is not expected: %d", index)
		} else if index >= replicas {
			// Service index out of range, which indicates that it is a scale in
			// reconciliation and pod index>=replica will be deleted later, so
			// we'd increase capacity of pod slice to collect.
			newServiceSlices := make([][]*v1.Service, index+1)
			copy(newServiceSlices, serviceSlices)
			serviceSlices = newServiceSlices
		}
		serviceSlices[index] = append(serviceSlices[index], service)
	}
	return serviceSlices
}

// reconcileServices checks and updates services for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting services.
func (jc *JobController) ReconcileServices(
	ctx context.Context,
	job metav1.Object,
	services []*v1.Service,
	rtype apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec) error {

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services, err := jc.FilterServicesForReplicaType(services, rt)
	if err != nil {
		return err
	}

	serviceSlices := jc.GetServiceSlices(services, replicas, commonutil.LoggerForReplica(job, rt))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			commonutil.LoggerForReplica(job, rt).Warningf("we have too many services for %s %d", rt, index)
		} else if len(serviceSlice) == 0 {
			commonutil.LoggerForReplica(job, rt).Infof("need to create new service: %s-%d", rt, index)
			err = jc.CreateNewService(ctx, job, rtype, spec, strconv.Itoa(index))
			if err != nil {
				return err
			}
		} else { /* len(serviceSlice) == 1 */
			service := serviceSlice[0]
			// Check if index is in valid range, otherwise we should scale in extra pods since
			// the expected replicas has been modified.
			if index >= replicas {
				if service.DeletionTimestamp == nil {
					jc.Recorder.Eventf(job.(runtime.Object), v1.EventTypeNormal, "DeleteService",
						"service %s/%s with index %v is out of expected replicas %v and should be deleted", service.Namespace, service.Name, index, replicas)
					if err = jc.ServiceControl.DeleteService(service.Namespace, service.Name, job.(runtime.Object)); err != nil {
						return err
					}
				}
			}
			if EnableHostNetwork(job) {
				hostPort, ok := GetHostNetworkPortFromContext(ctx, rt, strconv.Itoa(index))
				if ok && len(service.Spec.Ports) > 0 && service.Spec.Ports[0].TargetPort.IntVal != hostPort {
					commonutil.LoggerForReplica(job, rt).Infof("update target service: %s-%d, new port: %d",
						rt, index, hostPort)
					// Update service target port to latest container host port, because replicas may fail-over
					// and its host port changed, so we'd ensure that other replicas can reach it with correct
					// target port.
					newService := service.DeepCopy()
					newService.Spec.Ports[0].TargetPort = intstr.FromInt(int(hostPort))
					err = jc.patcher(service, newService)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// GetPortFromJob gets the port of job container.
func (jc *JobController) GetPortFromJob(spec *apiv1.ReplicaSpec) (int32, error) {
	containers := spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == jc.Controller.GetDefaultContainerName() {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == jc.Controller.GetDefaultContainerPortName() {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("failed to find the port")
}

// createNewService creates a new service for the given index and type.
func (jc *JobController) CreateNewService(ctx context.Context, job metav1.Object, rtype apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec, index string) error {

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	// Append ReplicaTypeLabel and ReplicaIndexLabel labels.
	labels := jc.GenLabels(job.GetName())
	labels[apiv1.ReplicaTypeLabel] = rt
	labels[apiv1.ReplicaIndexLabel] = index

	svcPort, err := jc.GetPortFromJob(spec)
	if err != nil {
		return err
	}
	targetPort := svcPort
	clusterIP := "None"

	if !features.KubeDLFeatureGates.Enabled(features.HostNetWithHeadlessSvc) && EnableHostNetwork(job) {
		// Communications between replicas use headless services by default, as for hostnetwork mode,
		// headless service can not forward traffic from one port to another, so we use normal service
		// when hostnetwork enabled.
		clusterIP = ""
		// retrieve selected host port with corresponding pods.
		hostPort, ok := GetHostNetworkPortFromContext(ctx, rt, index)
		if ok {
			targetPort = hostPort
		}
	}

	service := &v1.Service{
		Spec: v1.ServiceSpec{
			ClusterIP: clusterIP,
			Selector:  labels,
			Ports: []v1.ServicePort{
				{
					Name:       jc.Controller.GetDefaultContainerPortName(),
					Port:       svcPort,
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
		},
	}
	service.Labels = commonutil.MergeMap(service.Labels, labels)

	return jc.CreateService(job, rtype, service, index)
}

// CreateService creates a new common service with the given index and type.
func (jc *JobController) CreateService(job metav1.Object, rtype apiv1.ReplicaType,
	service *v1.Service, index string) error {
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	expectationServicesKey := GenExpectationServicesKey(jobKey, rt)
	err = jc.Expectations.ExpectCreations(expectationServicesKey, 1)
	if err != nil {
		return err
	}

	service.Name = commonutil.GenGeneralName(job.GetName(), rt, index)
	// Create OwnerReference.
	controllerRef := jc.GenOwnerReference(job)

	err = jc.ServiceControl.CreateServicesWithControllerRef(job.GetNamespace(), service, job.(runtime.Object), controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Service is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the service keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// service when the expectation expires.
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func (jc *JobController) AdoptAndClaimServices(job metav1.Object, serviceList *v1.ServiceList) ([]*v1.Service, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: jc.GenLabels(job.GetName()),
	})
	if err != nil {
		return nil, err
	}
	services := commonutil.ToServicePointerList(serviceList.Items)
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
	cm := NewServiceControllerRefManager(jc.ServiceControl, job, selector, jc.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimServices(services)
}
