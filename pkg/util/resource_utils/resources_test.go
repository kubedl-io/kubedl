package resource_utils

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func newResource(str string) resource.Quantity {
	v := resource.MustParse(str)
	val := resource.NewQuantity((&v).Value(), v.Format)
	return *val
}

func TestComputePodSpecResourceRequest(t *testing.T) {
	type args struct {
		spec *v1.PodSpec
	}
	tests := []struct {
		name string
		args args
		want v1.ResourceList
	}{
		{
			name: "no init",
			args: args{
				spec: &v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("100Mi"),
								},
							},
						},
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("200Mi"),
								},
							},
						},
					},
				},
			},
			want: v1.ResourceList{
				"mem": newResource("300Mi"),
			},
		}, {
			name: "init max",
			args: args{
				spec: &v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("350Mi"),
								},
							},
						},
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("200Mi"),
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("100Mi"),
								},
							},
						},
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("200Mi"),
								},
							},
						},
					},
				},
			},
			want: v1.ResourceList{
				"mem": resource.MustParse("350Mi"),
			},
		}, {
			name: "init",
			args: args{
				spec: &v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("150Mi"),
								},
							},
						},
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("200Mi"),
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("100Mi"),
								},
							},
						},
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"mem": resource.MustParse("300Mi"),
								},
								Requests: v1.ResourceList{
									"mem": resource.MustParse("200Mi"),
								},
							},
						},
					},
				},
			},
			want: v1.ResourceList{
				"mem": newResource("300Mi"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputePodSpecResourceRequest(tt.args.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ComputePodSpecResourceRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
