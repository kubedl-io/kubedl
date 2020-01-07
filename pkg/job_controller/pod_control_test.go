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

	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

var _ = Describe("PodControl", func() {

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Pod Control", func() {
		It("Should create successfully", func() {
			testName := "pod-name"

			podControl := PodControl{
				client:   k8sClient,
				recorder: &record.FakeRecorder{},
			}
			testJob := testutilv1.NewTestJob(1)

			podTemplate := testutilv1.NewTestReplicaSpecTemplate()
			podTemplate.Name = testName
			podTemplate.Namespace = testJob.Namespace
			podTemplate.Labels = testutilv1.GenLabels(testJob.Name)
			podTemplate.SetOwnerReferences([]metav1.OwnerReference{})

			// Make sure createReplica sends a POST to the apiserver with a pod from the controllers pod template
			err := podControl.CreatePods(testJob.Namespace, &podTemplate, testJob)
			Expect(err).Should(Succeed())

			expectedPod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    testutilv1.GenLabels(testJob.Name),
					Name:      testName,
					Namespace: testJob.Namespace,
				},
				Spec: podTemplate.Spec,
			}

			actualPod := &v1.Pod{}
			err = podControl.client.Get(context.Background(), types.NamespacedName{Namespace: testJob.Namespace, Name: testName}, actualPod)
			Expect(err).Should(Succeed())
			Expect(apiequality.Semantic.DeepDerivative(&expectedPod, actualPod)).Should(BeTrue())
		})
	})
})
