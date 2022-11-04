package core

import (
	"reflect"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator"

	"github.com/alibaba/kubedl/pkg/jobcoordinator/plugins"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/plugins/priority"
)

func TestUpdateScorePluginsList(t *testing.T) {
	testCases := []struct {
		name       string
		pluginList []jobcoordinator.ScorePlugin
		registry   plugins.Registry
		enables    []string
		expected   interface{}
	}{
		{
			name:       "update single score plugin with enable plugin",
			pluginList: []jobcoordinator.ScorePlugin{},
			registry:   plugins.NewPluginsRegistry(),
			enables:    []string{priority.Name},
			expected: []jobcoordinator.ScorePlugin{
				priority.New(nil, nil).(jobcoordinator.ScorePlugin),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := updatePluginList(nil, nil, &testCase.pluginList, testCase.registry, testCase.enables)
			if err != nil {
				t.Errorf("run updatePluginList failed, err: %v", err)
			}
			if !reflect.DeepEqual(testCase.expected, testCase.pluginList) {
				t.Errorf("unexpected pluginList, expected: %v, got: %v", testCase.expected, testCase.pluginList)
			}
		})
	}
}

func TestSelectQueueUnit(t *testing.T) {
	testCases := []struct {
		input  []jobcoordinator.QueueUnitScore
		output jobcoordinator.QueueUnitScore
	}{
		{
			input: []jobcoordinator.QueueUnitScore{
				{
					QueueUnit: &jobcoordinator.QueueUnit{
						Tenant:   "pai-dlc-default",
						Priority: pointer.Int32Ptr(5),
						Object: &trainingv1alpha1.TFJob{
							TypeMeta:   v1.TypeMeta{Kind: "TFJob"},
							ObjectMeta: v1.ObjectMeta{Name: "job-1", Namespace: "default"},
						},
					},
					Score: 5,
				},
				{
					QueueUnit: &jobcoordinator.QueueUnit{
						Tenant:   "pai-dlc-default",
						Priority: pointer.Int32Ptr(6),
						Object: &trainingv1alpha1.TFJob{
							TypeMeta:   v1.TypeMeta{Kind: "TFJob"},
							ObjectMeta: v1.ObjectMeta{Name: "job-2", Namespace: "default"},
						},
					},
					Score: 6,
				},
			},
			output: jobcoordinator.QueueUnitScore{
				QueueUnit: &jobcoordinator.QueueUnit{
					Tenant:   "pai-dlc-default",
					Priority: pointer.Int32Ptr(6),
					Object: &trainingv1alpha1.TFJob{
						TypeMeta:   v1.TypeMeta{Kind: "TFJob"},
						ObjectMeta: v1.ObjectMeta{Name: "job-2", Namespace: "default"},
					},
				},
				Score: 6,
			},
		},
		{
			input: []jobcoordinator.QueueUnitScore{
				{
					QueueUnit: &jobcoordinator.QueueUnit{
						Tenant:   "pai-dlc-default",
						Priority: pointer.Int32Ptr(5),
						Object: &trainingv1alpha1.TFJob{
							TypeMeta:   v1.TypeMeta{Kind: "TFJob"},
							ObjectMeta: v1.ObjectMeta{Name: "job-1", Namespace: "default"},
						},
					},
					Score: 5,
				},
				{
					QueueUnit: &jobcoordinator.QueueUnit{
						Tenant:   "pai-dlc-default",
						Priority: pointer.Int32Ptr(6),
						Object: &trainingv1alpha1.TFJob{
							TypeMeta:   v1.TypeMeta{Kind: "TFJob"},
							ObjectMeta: v1.ObjectMeta{Name: "job-2", Namespace: "default"},
						},
					},
					Score: 6,
				},
				{
					QueueUnit: &jobcoordinator.QueueUnit{
						Tenant:   "pai-dlc-default",
						Priority: pointer.Int32Ptr(7),
						Object: &trainingv1alpha1.TFJob{
							TypeMeta:   v1.TypeMeta{Kind: "TFJob"},
							ObjectMeta: v1.ObjectMeta{Name: "job-3", Namespace: "default"},
						},
					},
					Score: 7,
				},
			},
			output: jobcoordinator.QueueUnitScore{
				QueueUnit: &jobcoordinator.QueueUnit{
					Tenant:   "pai-dlc-default",
					Priority: pointer.Int32Ptr(7),
					Object: &trainingv1alpha1.TFJob{
						TypeMeta:   v1.TypeMeta{Kind: "TFJob"},
						ObjectMeta: v1.ObjectMeta{Name: "job-3", Namespace: "default"},
					},
				},
				Score: 7,
			},
		},
	}

	for _, testCase := range testCases {
		co := coordinator{}
		out := co.selectQueueUnit(testCase.input)
		if !reflect.DeepEqual(out, testCase.output) {
			t.Errorf("unexpected selected queue unit, expected: %+v, got: %+v", testCase.output, out)
		}
	}
}

var _ jobcoordinator.TenantPlugin = &emptyTenantPlugin{}

type emptyTenantPlugin struct{}

func (e emptyTenantPlugin) TenantName(qu *jobcoordinator.QueueUnit) string {
	return ""
}

func (e emptyTenantPlugin) Name() string {
	return ""
}
