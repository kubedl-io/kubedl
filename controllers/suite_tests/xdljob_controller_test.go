/*
Copyright 2019 The Alibaba Authors.

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

package suite_tests

import (
	"time"

	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	// +kubebuilder:scaffold:imports
)

const timeout = time.Second * 5

var _ = Describe("XDLJob Controller", func() {

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Job with schedule", func() {
		It("Should create successfully", func() {
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}

			instance := &xdlv1alpha1.XDLJob{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}

			// Create
			err := k8sClient.Create(context.Background(), instance)
			if apierrors.IsInvalid(err) {
				klog.Errorf("failed to create object, got an invalid object error: %v", err)
				return
			}
			// .NotTo(HaveOccurred())
			Expect(err).Should(Succeed())

			// Get
			By("Expecting created xdl job")
			Eventually(func() error {
				job := &xdlv1alpha1.XDLJob{}
				return k8sClient.Get(context.Background(), key, job)
			}, timeout).Should(BeNil())

			// Delete
			By("Deleting xdl job")
			Eventually(func() error {
				job := &xdlv1alpha1.XDLJob{}
				err := k8sClient.Get(context.Background(), key, job)
				if err != nil {
					return err
				}
				return k8sClient.Delete(context.Background(), job)
			}, timeout).Should(Succeed())
		})
	})
})
