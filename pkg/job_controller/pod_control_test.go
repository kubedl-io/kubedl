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
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	utiltesting "k8s.io/client-go/util/testing"
	"k8s.io/kubernetes/pkg/api/testapi"

	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
)

func TestCreatePods(t *testing.T) {
	ns := metav1.NamespaceDefault
	body := runtime.EncodeOrDie(testapi.Default.Codec(), &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "empty_pod"}})
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   200,
		ResponseBody: string(body),
	}
	testServer := httptest.NewServer(&fakeHandler)
	defer testServer.Close()

	podControl := PodControl{
		client:   fakeClient{},
		recorder: &record.FakeRecorder{},
	}

	testJob := testutilv1.NewTestJob(1)

	testName := "pod-name"
	podTemplate := testutilv1.NewTestReplicaSpecTemplate()
	podTemplate.Name = testName
	podTemplate.Labels = testutilv1.GenLabels(testJob.Name)
	podTemplate.SetOwnerReferences([]metav1.OwnerReference{})

	// Make sure createReplica sends a POST to the apiserver with a pod from the controllers pod template
	err := podControl.CreatePods(ns, &podTemplate, testJob)
	assert.NoError(t, err, "unexpected error: %v", err)

	expectedPod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: testutilv1.GenLabels(testJob.Name),
			Name:   testName,
		},
		Spec: podTemplate.Spec,
	}
	fakeHandler.ValidateRequest(t, testapi.Default.ResourcePath("pods", metav1.NamespaceDefault, ""), "POST", nil)
	var actualPod = &v1.Pod{}
	err = json.Unmarshal([]byte(fakeHandler.RequestBody), actualPod)
	assert.NoError(t, err, "unexpected error: %v", err)
	assert.True(t, apiequality.Semantic.DeepDerivative(&expectedPod, actualPod),
		"Body: %s", fakeHandler.RequestBody)
}
