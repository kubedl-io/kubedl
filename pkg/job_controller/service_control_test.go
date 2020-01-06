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
	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
)

var _ = Describe("ServiceControl", func() {

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Service Control", func() {
		It("Should create successfully", func() {
			testName := "service-name"

			svcControl := ServiceControl{
				client:   k8sClient,
				recorder: &record.FakeRecorder{},
			}
			testPortName := "test-port"
			testProtocol := v1.ProtocolTCP
			testPort := int32(8888)
			testTargetPort := intstr.IntOrString{IntVal: 8888}
			testJob := testutilv1.NewTestJob(1)

			service := testutilv1.NewBaseService(testName, testJob, nil)
			service.Spec.Ports = []v1.ServicePort{
				{
					Name:       testPortName,
					Protocol:   testProtocol,
					Port:       testPort,
					TargetPort: testTargetPort,
				},
			}
			service.SetOwnerReferences([]metav1.OwnerReference{})

			// Make sure createReplica sends a POST to the apiserver with a pod from the controllers pod template
			err := svcControl.CreateServices(testJob.Namespace, service, testJob)
			Expect(err).Should(Succeed())

			expectedService := v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    testutilv1.GenLabels(testJob.Name),
					Name:      testName,
					Namespace: testJob.Namespace,
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name:       testPortName,
							Protocol:   testProtocol,
							Port:       testPort,
							TargetPort: testTargetPort,
						},
					},
				},
			}

			actualService := &v1.Service{}
			err = svcControl.client.Get(context.Background(), types.NamespacedName{Namespace: testJob.Namespace, Name: testName}, actualService)
			Expect(err).Should(Succeed())
			Expect(apiequality.Semantic.DeepDerivative(&expectedService, actualService)).Should(BeTrue())
		})
	})

	Context("ServiceControl with controllerRef", func() {
		It("Should create successfully with controllerRef", func() {
			testName := "service-name-1"

			svcControl := ServiceControl{
				client:   k8sClient,
				recorder: &record.FakeRecorder{},
			}

			testJob := testutilv1.NewTestJob(1)

			service := testutilv1.NewBaseService(testName, testJob, nil)
			service.SetOwnerReferences([]metav1.OwnerReference{})
			testPortName := "test-port"
			testProtocol := v1.ProtocolTCP
			testPort := int32(8888)
			testTargetPort := intstr.IntOrString{IntVal: 8888}
			service.Spec.Ports = []v1.ServicePort{
				{
					Name:       testPortName,
					Protocol:   testProtocol,
					Port:       testPort,
					TargetPort: testTargetPort,
				},
			}

			ownerRef := testutilv1.GenOwnerReference(testJob)

			// Make sure createReplica sends a POST to the apiserver with a pod from the controllers pod template
			err := svcControl.CreateServicesWithControllerRef(testJob.Namespace, service, testJob, ownerRef)
			Expect(err).Should(Succeed())

			expectedService := v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Labels:          testutilv1.GenLabels(testJob.Name),
					Name:            testName,
					Namespace:       testJob.Namespace,
					OwnerReferences: []metav1.OwnerReference{*ownerRef},
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name:       testPortName,
							Protocol:   testProtocol,
							Port:       testPort,
							TargetPort: testTargetPort,
						},
					},
				},
			}
			actualService := &v1.Service{}
			err = svcControl.client.Get(context.Background(), types.NamespacedName{Namespace: testJob.Namespace, Name: testName}, actualService)
			fmt.Printf("actual: %+v\n", *actualService)
			fmt.Printf("expected: %+v\n", expectedService)
			Expect(err).Should(Succeed())
			Expect(apiequality.Semantic.DeepDerivative(&expectedService, actualService)).Should(BeTrue())
		})
	})
})
