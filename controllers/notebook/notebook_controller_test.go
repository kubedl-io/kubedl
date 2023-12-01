/*
Copyright 2021 The KubeDL Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/alibaba/kubedl/apis"
	notebookv1alpha1 "github.com/alibaba/kubedl/apis/notebook/v1alpha1"
)

func init() {
	// Enable klog which is used in dependencies
	_ = pflag.Set("logtostderr", "true")
	_ = pflag.Set("v", "10")
}

// 1. First reconcile: a notebook crd, check that notebook pod is created and its status as CREATED
// 2. Second reconcile: mark notebook pod is running and check notebook status as running
// 3. Third reconcile: mark notebook pod is finished and check notebook status as terminated
func TestNotebookController(t *testing.T) {
	scheme := runtime.NewScheme()
	// required to register all used api schemes
	_ = apis.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	notebook := createNotebook("notebook1")
	notebookName := types.NamespacedName{
		Namespace: "default",
		Name:      "notebook1",
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(notebook).Build()
	eventBroadcaster := record.NewBroadcaster()

	// reconcile the version
	notebookReconciler := &NotebookReconciler{
		Client:   fakeClient,
		Log:      ctrl.Log.WithName("controllers").WithName("NotebookController"),
		Scheme:   scheme,
		recorder: eventBroadcaster.NewRecorder(scheme, corev1.EventSource{Component: "notebook-controller"}),
	}

	// first reconcile
	request := reconcile.Request{NamespacedName: notebookName}
	_, _ = notebookReconciler.Reconcile(context.Background(), request)

	notebookPod := &corev1.Pod{}
	_ = notebookReconciler.Get(context.TODO(), types.NamespacedName{
		Namespace: "default",
		Name:      "nb-" + notebookName.Name,
	}, notebookPod)
	// the notebookPod is created
	assert.Equal(t, "nb-"+notebookName.Name, notebookPod.Name)
	assert.Equal(t, notebookv1alpha1.NotebookContainerName, notebookPod.Spec.Containers[0].Name)
	assert.Equal(t, notebookv1alpha1.NotebookPortName, notebookPod.Spec.Containers[0].Ports[0].Name)

	notebook = getNotebook(notebookReconciler, notebookName)
	assert.Equal(t, notebookv1alpha1.NotebookCreated, notebook.Status.Condition)

	notebookService := &corev1.Service{}
	_ = notebookReconciler.Get(context.TODO(), types.NamespacedName{
		Namespace: "default",
		Name:      "nb-" + notebookName.Name,
	}, notebookService)
	// the notebook service is created
	assert.Equal(t, "nb-"+notebookName.Name, notebookService.Name)

	ingress := &v1.Ingress{}
	_ = notebookReconciler.Get(context.TODO(), types.NamespacedName{
		Namespace: "default",
		Name:      "nb-" + notebookName.Name,
	}, ingress)
	// the notebook ingress is created
	assert.Equal(t, "nb-"+notebookName.Name, ingress.Name)

	notebookPod.Status.Phase = corev1.PodRunning
	_ = notebookReconciler.Status().Update(context.TODO(), notebookPod)
	// now, notebook pod is running,
	// 2nd reconcile
	_, _ = notebookReconciler.Reconcile(context.Background(), request)
	// notebook is running
	notebook = getNotebook(notebookReconciler, notebookName)
	assert.Equal(t, notebookv1alpha1.NotebookRunning, notebook.Status.Condition)

	// pod terminated
	notebookPod.Status.Phase = corev1.PodSucceeded
	_ = notebookReconciler.Status().Update(context.TODO(), notebookPod)
	// 3rd reconcile
	_, _ = notebookReconciler.Reconcile(context.Background(), request)
	// notebook is termniated
	notebook = getNotebook(notebookReconciler, notebookName)
	assert.Equal(t, notebookv1alpha1.NotebookTerminated, notebook.Status.Condition)
}

func getNotebook(reconciler *NotebookReconciler, name types.NamespacedName) *notebookv1alpha1.Notebook {
	notebook := &notebookv1alpha1.Notebook{}
	_ = reconciler.Get(context.TODO(), name, notebook)
	return notebook
}

func createNotebook(nbName string) *notebookv1alpha1.Notebook {
	notebook := &notebookv1alpha1.Notebook{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "default",
			Name:        nbName,
			UID:         "9423255b-1123-11e7-af6a-28d2447dc82b",
			Labels:      make(map[string]string, 0),
			Annotations: make(map[string]string, 0),
		},
		Spec: notebookv1alpha1.NotebookSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: notebookv1alpha1.NotebookContainerName,
							Ports: []corev1.ContainerPort{
								{
									Name:          notebookv1alpha1.NotebookPortName,
									ContainerPort: notebookv1alpha1.NotebookDefaultPort,
								},
							},
						},
					},
				},
			},
		},
		Status: notebookv1alpha1.NotebookStatus{},
	}
	return notebook
}
