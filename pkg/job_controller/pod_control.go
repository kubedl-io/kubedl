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
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reasons for pod events
const (
	// FailedCreatePodReason is added in an event and in a replica set condition
	// when a pod for a replica set is failed to be created.
	FailedCreatePodReason = "FailedCreatePod"
	// SuccessfulCreatePodReason is added in an event when a pod for a replica set
	// is successfully created.
	SuccessfulCreatePodReason = "SuccessfulCreatePod"
	// FailedDeletePodReason is added in an event and in a replica set condition
	// when a pod for a replica set is failed to be deleted.
	FailedDeletePodReason = "FailedDeletePod"
	// SuccessfulDeletePodReason is added in an event when a pod for a replica set
	// is successfully deleted.
	SuccessfulDeletePodReason = "SuccessfulDeletePod"
)

// PodControl is the default implementation of PodControlInterface.
type PodControl struct {
	client   client.Client
	recorder record.EventRecorder
}

var _ controller.PodControlInterface = &PodControl{}

func NewPodControl(client client.Client, recorder record.EventRecorder) controller.PodControlInterface {
	return &PodControl{
		client:   client,
		recorder: recorder,
	}
}

func getPodsLabelSet(template *v1.PodTemplateSpec) labels.Set {
	desiredLabels := make(labels.Set)
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}
	return desiredLabels
}

func getPodsFinalizers(template *v1.PodTemplateSpec) []string {
	desiredFinalizers := make([]string, len(template.Finalizers))
	copy(desiredFinalizers, template.Finalizers)
	return desiredFinalizers
}

func getPodsAnnotationSet(template *v1.PodTemplateSpec) labels.Set {
	desiredAnnotations := make(labels.Set)
	for k, v := range template.Annotations {
		desiredAnnotations[k] = v
	}
	return desiredAnnotations
}

func getPodsOwnerReferences(template *v1.PodTemplateSpec) []metav1.OwnerReference {
	desiredOwnerReferences := make([]metav1.OwnerReference, len(template.OwnerReferences))
	copy(desiredOwnerReferences, template.OwnerReferences)
	return desiredOwnerReferences
}

func (r PodControl) CreatePods(namespace string, template *v1.PodTemplateSpec, object runtime.Object) error {
	return r.createPods("", namespace, template, object, nil)
}

func (r PodControl) CreatePodsWithControllerRef(namespace string, template *v1.PodTemplateSpec, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createPods("", namespace, template, controllerObject, controllerRef)
}

func (r PodControl) CreatePodsOnNode(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createPods(nodeName, namespace, template, object, controllerRef)
}

func (r PodControl) PatchPod(namespace, name string, data []byte) error {
	pod := &v1.Pod{}
	if err := r.client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, pod); err != nil {
		return err
	}
	return r.client.Patch(context.Background(), pod, client.ConstantPatch(types.StrategicMergePatchType, data))
}

func GetPodFromTemplate(template *v1.PodTemplateSpec, parentObject runtime.Object, controllerRef *metav1.OwnerReference) (*v1.Pod, error) {
	desiredLabels := getPodsLabelSet(template)
	desiredFinalizers := getPodsFinalizers(template)
	desiredAnnotations := getPodsAnnotationSet(template)
	desiredOwnerReferences := getPodsOwnerReferences(template)

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          desiredLabels,
			Annotations:     desiredAnnotations,
			Name:            template.Name,
			Finalizers:      desiredFinalizers,
			OwnerReferences: desiredOwnerReferences,
		},
	}
	if controllerRef != nil {
		pod.OwnerReferences = append(pod.OwnerReferences, *controllerRef)
	}
	pod.Spec = *template.Spec.DeepCopy()
	return pod, nil
}

func (r PodControl) createPods(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	pod, err := GetPodFromTemplate(template, object, controllerRef)
	if err != nil {
		return err
	}
	if len(nodeName) != 0 {
		pod.Spec.NodeName = nodeName
	}
	pod.Namespace = namespace
	if labels.Set(pod.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create pods, no labels")
	}
	if err := r.client.Create(context.Background(), pod); err != nil {
		r.recorder.Eventf(object, v1.EventTypeWarning, FailedCreatePodReason, "Error creating: %v", err)
		return err
	} else {
		accessor, err := meta.Accessor(object)
		if err != nil {
			glog.Errorf("parentObject does not have ObjectMeta, %v", err)
			return nil
		}
		glog.V(4).Infof("Controller %v created pod %v", accessor.GetName(), pod.Name)
		r.recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreatePodReason, "Created pod: %v", pod.Name)
	}
	return nil
}

func (r PodControl) DeletePod(namespace string, name string, object runtime.Object) error {
	pod := &v1.Pod{}
	err := r.client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, pod)
	if err != nil {
		return err
	}
	if pod.DeletionTimestamp != nil {
		glog.V(3).Infof("pod %s/%s is terminating, skip deleting", pod.Namespace, pod.Name)
		return nil
	}
	accessor, err := meta.Accessor(object)
	if err != nil {
		return fmt.Errorf("object does not have ObjectMeta, %v", err)
	}
	glog.V(2).Infof("Controller %v deleting pod %v/%v", accessor.GetName(), namespace, name)
	if err = r.client.Delete(context.Background(), pod); err != nil {
		r.recorder.Eventf(object, v1.EventTypeWarning, FailedDeletePodReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete pods: %v", err)
	} else {
		r.recorder.Eventf(object, v1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted pod: %v", name)
	}
	return nil
}
