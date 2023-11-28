package quota

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/helper"
)

func TestQuota_Filter(t *testing.T) {
	testCases := []struct {
		name           string
		q              *corev1.ResourceQuota
		qu             *jobcoordinator.QueueUnit
		expectedStatus jobcoordinator.Status
	}{
		{
			name: "quota filter with default tenant queue and no resources limit",
			q: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tenant-1",
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: map[corev1.ResourceName]resource.Quantity{}},
				Status: corev1.ResourceQuotaStatus{
					Used: map[corev1.ResourceName]resource.Quantity{}},
			},
			qu:             newTFQueueUnit("job-1", DefaultTenantName, "1", "1Gi", "1", 1),
			expectedStatus: jobcoordinator.NewStatus(jobcoordinator.Success),
		},
		{
			name: "quota filter custom tenant name and cpu/memory enough, tfjob with 1 replica ",
			q: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tenant-2",
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("10"),
						corev1.ResourceMemory: resource.MustParse("20Gi"),
					}},
				Status: corev1.ResourceQuotaStatus{
					Used: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("5Gi"),
					}},
			},
			qu:             newTFQueueUnit("job-2", "tenant-2", "1", "1Gi", "1", 1),
			expectedStatus: jobcoordinator.NewStatus(jobcoordinator.Success),
		},
		{
			name: "quota filter custom tenant name and cpu/memory not enough, tfjob with 6 replica ",
			q: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tenant-3",
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("10"),
						corev1.ResourceMemory: resource.MustParse("20Gi"),
					}},
				Status: corev1.ResourceQuotaStatus{
					Used: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("5Gi"),
					}},
			},
			qu:             newTFQueueUnit("job-3", "tenant-3", "1", "1Gi", "1", 6),
			expectedStatus: jobcoordinator.NewStatus(jobcoordinator.Wait),
		},
		{
			name: "quota filter custom tenant name and cpu/memory/gpu enough, tfjob with 1 replica ",
			q: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tenant-4",
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:      resource.MustParse("10"),
						corev1.ResourceMemory:   resource.MustParse("20Gi"),
						apiv1.ResourceNvidiaGPU: resource.MustParse("5"),
					}},
				Status: corev1.ResourceQuotaStatus{
					Used: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("5Gi"),
					}},
			},
			qu:             newTFQueueUnit("job-4", "tenant-4", "1", "1Gi", "1", 1),
			expectedStatus: jobcoordinator.NewStatus(jobcoordinator.Success),
		},
		{
			name: "quota filter custom tenant name and cpu/memory enough but gpu runs out, tfjob with 6 replica",
			q: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tenant-5",
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:      resource.MustParse("10"),
						corev1.ResourceMemory:   resource.MustParse("20Gi"),
						apiv1.ResourceNvidiaGPU: resource.MustParse("5"),
					}},
				Status: corev1.ResourceQuotaStatus{
					Used: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("5Gi"),
					}},
			},
			qu:             newTFQueueUnit("job-5", "tenant-5", "1", "1Gi", "1", 6),
			expectedStatus: jobcoordinator.NewStatus(jobcoordinator.Wait),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_ = corev1.AddToScheme(scheme.Scheme)
			q := quota{
				client:   fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				recorder: record.NewFakeRecorder(10),
			}

			used := testCase.q.Status.DeepCopy()
			if err := q.client.Create(context.Background(), testCase.q); err != nil {
				t.Error(err)
			}

			testCase.q.Status = *used
			if err := q.client.Update(context.Background(), testCase.q); err != nil {
				t.Error(err)
			}

			status := q.Filter(context.Background(), testCase.qu)
			if status.Code() != testCase.expectedStatus.Code() {
				t.Errorf("unexpected filter status, expected: %v, got: %v", testCase.expectedStatus, status)
			}
		})
	}
}

func newTFQueueUnit(name, tenant, cpu, mem, gpu string, replicas int) *jobcoordinator.QueueUnit {
	tf := &trainingv1alpha1.TFJob{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: trainingv1alpha1.TFJobSpec{
			TFReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
				trainingv1alpha1.TFReplicaTypeWorker: {
					Replicas: pointer.Int32Ptr(int32(replicas)),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "tensorflow",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:      resource.MustParse(cpu),
											corev1.ResourceMemory:   resource.MustParse(mem),
											apiv1.ResourceNvidiaGPU: resource.MustParse(gpu),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	qu := helper.ToQueueUnit(tf, tf.Spec.TFReplicaSpecs, &tf.Status, tf.Spec.SchedulingPolicy)
	qu.Tenant = tenant
	return qu
}
