package suite_tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	inference "github.com/alibaba/kubedl/apis/inference/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

var _ = Describe("ElasticBatchJob Controller", func() {

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

			instance := &inference.ElasticBatchJob{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}

			// Create
			err := k8sClient.Create(context.Background(), instance)
			if apierrors.IsInvalid(err) {
				klog.Errorf("failed to create object, got an invalid object error: %v", err)
				return
			}
			// .NotTo(HaveOccurred())
			Expect(err).Should(Succeed())

			// Get
			By("Expecting created elasticbatch job")
			Eventually(func() error {
				job := &inference.ElasticBatchJob{}
				return k8sClient.Get(context.Background(), key, job)
			}, timeout).Should(BeNil())

			// Delete
			By("Deleting elasticbatch job")
			Eventually(func() error {
				job := &inference.ElasticBatchJob{}
				err := k8sClient.Get(context.Background(), key, job)
				if err != nil {
					return err
				}
				return k8sClient.Delete(context.Background(), job)
			}, timeout).Should(Succeed())
		})
	})
})
