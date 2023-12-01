package priority

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator"
)

func TestPriority_Score(t *testing.T) {
	testCases := []struct {
		name          string
		schedPolicy   apiv1.SchedulingPolicy
		priorityClass *schedulingv1.PriorityClass
		expectedValue int64
	}{
		{
			name:          "test-1",
			schedPolicy:   apiv1.SchedulingPolicy{},
			expectedValue: 0,
		},
		{
			name:          "test-2",
			schedPolicy:   apiv1.SchedulingPolicy{Priority: pointer.Int32Ptr(100)},
			expectedValue: 100,
		},
		{
			name:          "test-2",
			schedPolicy:   apiv1.SchedulingPolicy{Priority: pointer.Int32Ptr(100)},
			expectedValue: 100,
		},
		{
			name:        "test-3",
			schedPolicy: apiv1.SchedulingPolicy{PriorityClassName: "test"},
			priorityClass: &schedulingv1.PriorityClass{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Value:      200,
			},
			expectedValue: 200,
		},
		{
			name:        "test-4",
			schedPolicy: apiv1.SchedulingPolicy{Priority: pointer.Int32Ptr(100)},
			priorityClass: &schedulingv1.PriorityClass{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Value:      200,
			},
			expectedValue: 100,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_ = schedulingv1.AddToScheme(scheme.Scheme)
			prio := priority{clientSet: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()}
			if testCase.priorityClass != nil {
				_ = prio.clientSet.Create(context.Background(), testCase.priorityClass)
			}
			qu := newQueueUnit(testCase.name, testCase.schedPolicy)
			val, status := prio.Score(context.Background(), qu)

			if !status.IsSuccess() {
				t.Errorf("unsuccess status")
			}
			if val != testCase.expectedValue {
				t.Errorf("unexpected priority value, expected: %v, got: %v", testCase.expectedValue, val)
			}
		})
	}
}

func newQueueUnit(name string, schedPolicy apiv1.SchedulingPolicy) *jobcoordinator.QueueUnit {
	p := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: name,
	}}
	return &jobcoordinator.QueueUnit{
		Object:      p,
		SchedPolicy: &schedPolicy,
	}
}
