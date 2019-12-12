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
	"github.com/golang/glog"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

const (
	FailedCreateServiceReason     = "FailedCreateService"
	SuccessfulCreateServiceReason = "SuccessfulCreateService"
	FailedDeleteServiceReason     = "FailedDeleteService"
	SuccessfulDeleteServiceReason = "SuccessfulDeleteService"
)

// ServiceControlInterface is an interface that knows how to add or delete Services
// created as an interface to allow testing.
type ServiceControlInterface interface {
	// CreateServices creates new Services according to the spec.
	CreateServices(namespace string, service *v1.Service, object runtime.Object) error
	// CreateServicesWithControllerRef creates new services according to the spec, and sets object as the service's controller.
	CreateServicesWithControllerRef(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error
	// PatchService patches the service.
	PatchService(namespace, name string, data []byte) error
	// DeleteService deletes the service identified by serviceID.
	DeleteService(namespace, serviceID string, object runtime.Object) error
}

func validateControllerRef(controllerRef *metav1.OwnerReference) error {
	if controllerRef == nil {
		return fmt.Errorf("controllerRef is nil")
	}
	if len(controllerRef.APIVersion) == 0 {
		return fmt.Errorf("controllerRef has empty APIVersion")
	}
	if len(controllerRef.Kind) == 0 {
		return fmt.Errorf("controllerRef has empty Kind")
	}
	if controllerRef.Controller == nil || !*controllerRef.Controller {
		return fmt.Errorf("controllerRef.Controller is not set to true")
	}
	if controllerRef.BlockOwnerDeletion == nil || !*controllerRef.BlockOwnerDeletion {
		return fmt.Errorf("controllerRef.BlockOwnerDeletion is not set")
	}
	return nil
}

// ServiceControl is the default implementation of ServiceControlInterface.
type ServiceControl struct {
	client   client.Client
	recorder record.EventRecorder
}

func NewServiceControl(client client.Client, recorder record.EventRecorder) *ServiceControl {
	return &ServiceControl{
		client:   client,
		recorder: recorder,
	}
}

func (r ServiceControl) PatchService(namespace, name string, data []byte) error {
	service := &v1.Service{}
	if err := r.client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, service); err != nil {
		return err
	}
	return r.client.Patch(context.Background(), service, client.ConstantPatch(types.StrategicMergePatchType, data))
}

func (r ServiceControl) CreateServices(namespace string, service *v1.Service, object runtime.Object) error {
	return r.createServices(namespace, service, object, nil)
}

func (r ServiceControl) CreateServicesWithControllerRef(namespace string, service *v1.Service, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createServices(namespace, service, controllerObject, controllerRef)
}

func (r ServiceControl) createServices(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	if labels.Set(service.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create Services, no labels")
	}
	serviceWithOwner, err := getServiceFromTemplate(service, object, controllerRef)
	if err != nil {
		r.recorder.Eventf(object, v1.EventTypeWarning, FailedCreateServiceReason, "Error creating: %v", err)
		return fmt.Errorf("unable to create services: %v", err)
	}
	err = r.client.Create(context.Background(), serviceWithOwner)
	if err != nil {
		r.recorder.Eventf(object, v1.EventTypeWarning, FailedCreateServiceReason, "Error creating: %v", err)
		return fmt.Errorf("unable to create services: %v", err)
	}

	accessor, err := meta.Accessor(object)
	if err != nil {
		log.Errorf("parentObject does not have ObjectMeta, %v", err)
		return nil
	}
	log.Infof("Controller %v created service %v", accessor.GetName(), serviceWithOwner.Name)
	r.recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreateServiceReason, "Created service: %v", serviceWithOwner.Name)

	return nil
}

// DeleteService deletes the service identified by name.
func (r ServiceControl) DeleteService(namespace, name string, object runtime.Object) error {
	service := &v1.Service{}
	err := r.client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, service)
	if err != nil {
		return err
	}
	if service.DeletionTimestamp != nil {
		glog.V(3).Infof("service %s/%s is terminating, skip deleting", service.Namespace, service.Name)
		return nil
	}
	accessor, err := meta.Accessor(object)
	if err != nil {
		return fmt.Errorf("object does not have ObjectMeta, %v", err)
	}
	log.Infof("Controller %v deleting service %v/%v", accessor.GetName(), namespace, name)
	if err := r.client.Delete(context.Background(), service); err != nil {
		r.recorder.Eventf(object, v1.EventTypeWarning, FailedDeleteServiceReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete service: %v", err)
	} else {
		r.recorder.Eventf(object, v1.EventTypeNormal, SuccessfulDeleteServiceReason, "Deleted service: %v", name)
	}
	return nil
}

func getServiceFromTemplate(template *v1.Service, parentObject runtime.Object, controllerRef *metav1.OwnerReference) (*v1.Service, error) {
	service := template.DeepCopy()
	if controllerRef != nil {
		service.OwnerReferences = append(service.OwnerReferences, *controllerRef)
	}
	return service, nil
}

type FakeServiceControl struct {
	sync.Mutex
	Templates         []v1.Service
	ControllerRefs    []metav1.OwnerReference
	DeleteServiceName []string
	Patches           [][]byte
	Err               error
	CreateLimit       int
	CreateCallCount   int
}

var _ ServiceControlInterface = &FakeServiceControl{}

func (f *FakeServiceControl) PatchService(namespace, name string, data []byte) error {
	f.Lock()
	defer f.Unlock()
	f.Patches = append(f.Patches, data)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) CreateServices(namespace string, service *v1.Service, object runtime.Object) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("not creating service, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *service)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) CreateServicesWithControllerRef(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("not creating service, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *service)
	f.ControllerRefs = append(f.ControllerRefs, *controllerRef)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) DeleteService(namespace string, serviceID string, object runtime.Object) error {
	f.Lock()
	defer f.Unlock()
	f.DeleteServiceName = append(f.DeleteServiceName, serviceID)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) Clear() {
	f.Lock()
	defer f.Unlock()
	f.DeleteServiceName = []string{}
	f.Templates = []v1.Service{}
	f.ControllerRefs = []metav1.OwnerReference{}
	f.Patches = [][]byte{}
	f.CreateLimit = 0
	f.CreateCallCount = 0
}
