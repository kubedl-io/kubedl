/*
Copyright 2021 The Alibaba Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/alibaba/kubedl/apis"
	"github.com/alibaba/kubedl/apis/model/v1alpha1"
)

func init() {
	// Enable klog which is used in dependencies
	_ = pflag.Set("logtostderr", "true")
	_ = pflag.Set("v", "10")
}

func TestCreateModelVersionWithLocalStorage(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = apis.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	version := createModelVersion("model1", "model-v1")

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(version).Build()

	// reconcile the version
	versionReconciler := &ModelVersionReconciler{
		Client: fakeClient,
		Log:    ctrl.Log.WithName("controllers").WithName("ModelVersion"),

		Scheme: scheme,
	}
	versionRequest := reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: "default",
		Name:      "model-v1",
	}}
	_, _ = versionReconciler.Reconcile(context.Background(), versionRequest)

	model := &v1alpha1.Model{}
	_ = versionReconciler.Get(context.TODO(), types.NamespacedName{
		Namespace: versionRequest.Namespace,
		Name:      "model1",
	}, model)
	// the model is created
	assert.Equal(t, "model1", model.Name)

	pvc := &corev1.PersistentVolumeClaim{}
	_ = versionReconciler.Get(context.TODO(), types.NamespacedName{
		Namespace: versionRequest.Namespace,
		Name:      GetModelVersionPVClaimNameByNode(version.Name, version.Spec.Storage.LocalStorage.NodeName),
	}, pvc)

	// bound the claim
	pvc.Status.Phase = corev1.ClaimBound
	_ = versionReconciler.Update(context.TODO(), pvc)
	_, _ = versionReconciler.Reconcile(context.Background(), versionRequest)

	// check the img build pod exists
	imgBuildPodName := GetBuildImagePodName(version.Spec.ModelName, string(version.UID[:5]))
	imgBuildPod := &corev1.Pod{}
	_ = versionReconciler.Get(context.TODO(), types.NamespacedName{
		Namespace: versionRequest.Namespace,
		Name:      imgBuildPodName,
	}, imgBuildPod)
	// check image build pod is generated
	assert.Equal(t, imgBuildPodName, imgBuildPod.Name)

	imgBuildPod.Status.Phase = corev1.PodSucceeded
	_ = versionReconciler.Status().Update(context.TODO(), imgBuildPod)
	_, _ = versionReconciler.Reconcile(context.Background(), versionRequest)
	_ = versionReconciler.Get(context.TODO(), types.NamespacedName{
		Namespace: version.Namespace,
		Name:      version.Name,
	}, version)
	assert.Equal(t, v1alpha1.ImageBuildSucceeded, version.Status.ImageBuildPhase)
}

func createModelVersion(modelName, versionName string) *v1alpha1.ModelVersion {
	modelVersion := &v1alpha1.ModelVersion{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "default",
			Name:        versionName,
			UID:         "9423255b-4600-11e7-af6a-28d2447dc82b",
			Labels:      make(map[string]string, 0),
			Annotations: make(map[string]string, 0),
		},
		Spec: v1alpha1.ModelVersionSpec{
			ModelName: modelName,
			CreatedBy: "user1",
			Storage: &v1alpha1.Storage{
				LocalStorage: &v1alpha1.LocalStorage{
					Path:     "/tmp/model",
					NodeName: "localhost",
				},
			},
		},
		Status: v1alpha1.ModelVersionStatus{},
	}
	return modelVersion
}
