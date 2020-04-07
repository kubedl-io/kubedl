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

package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/alibaba/kubedl/api"
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var k8sClient client.Client

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(func(options *zap.Options) {
		options.DestWritter = GinkgoWriter
		options.Development = true
	}))

	By("bootstrapping test environment")

	err := api.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient = fake.NewFakeClientWithScheme(scheme.Scheme)
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 30)

var _ = Describe("PodControl", func() {
	It("Running status counting", func() {
		now := v1.Now()
		toCreate := tfv1.TFJob{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: tfv1.TFJobSpec{TFReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{}},
			Status: apiv1.JobStatus{
				ReplicaStatuses: map[apiv1.ReplicaType]*apiv1.ReplicaStatus{},
				StartTime:       &now,
			},
		}
		_ = util.UpdateJobConditions(&toCreate.Status, apiv1.JobCreated, "", "")
		pending, err := JobStatusCounter(tfv1.Kind, k8sClient, func(status apiv1.JobStatus) bool {
			return util.IsCreated(status) && len(status.Conditions) == 1
		})
		Expect(err).Should(Succeed())
		Expect(pending).Should(Equal(int32(0)))

		err = k8sClient.Create(context.Background(), &toCreate)
		Expect(err).Should(Succeed())
		pending, err = JobStatusCounter(tfv1.Kind, k8sClient, func(status apiv1.JobStatus) bool {
			return util.IsCreated(status) && len(status.Conditions) == 1
		})
		Expect(err).Should(Succeed())
		Expect(pending).Should(Equal(int32(1)))

		fetched := &tfv1.TFJob{}
		err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "foo", Namespace: "default"}, fetched)
		Expect(err).Should(Succeed())

		err = k8sClient.Delete(context.Background(), fetched)
		Expect(err).Should(Succeed())
	})

	It("Pending status counting", func() {
		now := v1.Now()
		toCreate := tfv1.TFJob{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: tfv1.TFJobSpec{TFReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{}},
			Status: apiv1.JobStatus{
				ReplicaStatuses: map[apiv1.ReplicaType]*apiv1.ReplicaStatus{},
				StartTime:       &now,
			},
		}
		_ = util.UpdateJobConditions(&toCreate.Status, apiv1.JobCreated, "", "")
		_ = util.UpdateJobConditions(&toCreate.Status, apiv1.JobRunning, "", "")
		running, err := JobStatusCounter(tfv1.Kind, k8sClient, util.IsRunning)
		Expect(err).Should(Succeed())
		Expect(running).Should(Equal(int32(0)))

		err = k8sClient.Create(context.Background(), &toCreate)
		Expect(err).Should(Succeed())
		running, err = JobStatusCounter(tfv1.Kind, k8sClient, util.IsRunning)
		Expect(err).Should(Succeed())
		Expect(running).Should(Equal(int32(1)))

		fetched := &tfv1.TFJob{}
		err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "foo", Namespace: "default"}, fetched)
		Expect(err).Should(Succeed())

		err = k8sClient.Delete(context.Background(), fetched)
		Expect(err).Should(Succeed())
	})
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	gexec.KillAndWait(5 * time.Second)
})
