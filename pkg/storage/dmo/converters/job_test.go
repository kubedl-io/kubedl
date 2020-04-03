/*
Copyright 2020 The Alibaba Authors.

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

package converters

import (
	"reflect"
	"testing"

	pytorchv1 "github.com/alibaba/kubedl/api/pytorch/v1"
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
	xdlv1alpha1 "github.com/alibaba/kubedl/api/xdl/v1alpha1"
	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/util"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestConvertJobToDMOJob(t *testing.T) {
	type args struct {
		job    metav1.Object
		region string
	}
	tests := []struct {
		name    string
		args    args
		want    *dmo.Job
		wantErr bool
	}{
		{
			name: "tfjob with created status",
			args: args{
				job: &tfv1.TFJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "tfjob-test",
						Namespace:         testNamespace,
						UID:               "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
						ResourceVersion:   "3",
						CreationTimestamp: metav1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					},
					Status: apiv1.JobStatus{
						StartTime: &metav1.Time{Time: testTime("2019-02-11T12:27:00Z")},
					},
					Spec: tfv1.TFJobSpec{
						TFReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
							"Worker": {
								Template: v1.PodTemplateSpec{
									Spec: v1.PodSpec{
										Containers: []v1.Container{
											{
												Image: testImage,
												Resources: v1.ResourceRequirements{Requests: v1.ResourceList{
													"cpu":    resource.MustParse("1"),
													"memory": resource.MustParse("1Gi"),
												},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				region: testRegion,
			},
			want: &dmo.Job{
				Name:         "tfjob-test",
				JobID:        "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
				Kind:         tfv1.Kind,
				Status:       apiv1.JobCreated,
				Namespace:    testNamespace,
				DeployRegion: pointer.StringPtr(testRegion),
				Tenant:       pointer.StringPtr(""),
				Owner:        pointer.StringPtr(""),
				Deleted:      util.IntPtr(0),
				IsInEtcd:     util.IntPtr(1),
				Version:      "3",
				GmtCreated:   testTime("2019-02-10T12:27:00Z"),
				Resources:    `{"Worker":{"resources":{"requests":{"cpu":"1","memory":"1Gi"}},"replicas":0}}`,
			},
		}, {
			name: "tfjob with region",
			args: args{
				job: &tfv1.TFJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "tfjob-test",
						Namespace:         testNamespace,
						UID:               "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
						ResourceVersion:   "3",
						CreationTimestamp: metav1.Time{Time: testTime("2019-02-10T12:27:00Z")},
						Annotations: map[string]string{
							apiv1.AnnotationTenancyInfo: `{"tenant":"foo","user":"bar","idc":"test-idc","region":"test-region"}`,
						},
					},
					Status: apiv1.JobStatus{
						CompletionTime: &metav1.Time{Time: testTime("2019-02-11T12:27:00Z")},
						Conditions:     []apiv1.JobCondition{{Type: apiv1.JobSucceeded}},
					},
					Spec: tfv1.TFJobSpec{
						TFReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
							"Worker": {
								Replicas: pointer.Int32Ptr(1),
								Template: v1.PodTemplateSpec{
									Spec: v1.PodSpec{
										Containers: []v1.Container{
											{
												Name:  testMainContainerName,
												Image: testImage,
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
												},
											},
											{
												Name:  "sidecar",
												Image: testImage,
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &dmo.Job{
				Name:         "tfjob-test",
				JobID:        "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
				Kind:         tfv1.Kind,
				Status:       apiv1.JobSucceeded,
				Namespace:    testNamespace,
				DeployRegion: pointer.StringPtr(testRegion),
				Tenant:       pointer.StringPtr("foo"),
				Owner:        pointer.StringPtr("bar"),
				Deleted:      util.IntPtr(0),
				IsInEtcd:     util.IntPtr(1),
				Version:      "3",
				GmtCreated:   testTime("2019-02-10T12:27:00Z"),
				GmtFinished:  testTimePtr("2019-02-11T12:27:00Z"),
				Resources:    `{"Worker":{"resources":{"requests":{"cpu":"2","memory":"2Gi"}},"replicas":1}}`,
			},
		}, {
			name: "xdljob with status succeed",
			args: args{
				job: &xdlv1alpha1.XDLJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "xdljob-test",
						Namespace:         testNamespace,
						UID:               "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
						ResourceVersion:   "3",
						CreationTimestamp: metav1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					},
					Status: apiv1.JobStatus{
						CompletionTime: &metav1.Time{Time: testTime("2019-02-11T12:27:00Z")},
						Conditions:     []apiv1.JobCondition{{Type: apiv1.JobSucceeded}},
					},
					Spec: xdlv1alpha1.XDLJobSpec{
						XDLReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
							"Master": {
								Replicas: pointer.Int32Ptr(1),
								Template: v1.PodTemplateSpec{
									Spec: v1.PodSpec{
										Containers: []v1.Container{
											{
												Name:  testMainContainerName,
												Image: testImage,
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &dmo.Job{
				Name:        "xdljob-test",
				JobID:       "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
				Kind:        xdlv1alpha1.Kind,
				Status:      apiv1.JobSucceeded,
				Namespace:   testNamespace,
				Tenant:      pointer.StringPtr(""),
				Owner:       pointer.StringPtr(""),
				Deleted:     util.IntPtr(0),
				IsInEtcd:    util.IntPtr(1),
				Version:     "3",
				GmtCreated:  testTime("2019-02-10T12:27:00Z"),
				GmtFinished: testTimePtr("2019-02-11T12:27:00Z"),
				Resources:   `{"Master":{"resources":{"requests":{"cpu":"1","memory":"1Gi"}},"replicas":1}}`,
			},
		}, {
			name: "pytorchjob with succeed status",
			args: args{
				job: &pytorchv1.PyTorchJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pytorchjob-test",
						Namespace:         testNamespace,
						UID:               "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
						ResourceVersion:   "3",
						CreationTimestamp: metav1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					},
					Status: apiv1.JobStatus{
						CompletionTime: &metav1.Time{Time: testTime("2019-02-11T12:27:00Z")},
						Conditions:     []apiv1.JobCondition{{Type: apiv1.JobSucceeded}},
					},
					Spec: pytorchv1.PyTorchJobSpec{
						PyTorchReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
							"Worker": {
								Replicas: pointer.Int32Ptr(1),
								Template: v1.PodTemplateSpec{
									Spec: v1.PodSpec{
										Containers: []v1.Container{
											{
												Name:  testMainContainerName,
												Image: testImage,
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &dmo.Job{
				Name:        "pytorchjob-test",
				JobID:       "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
				Kind:        pytorchv1.Kind,
				Status:      apiv1.JobSucceeded,
				Namespace:   testNamespace,
				Tenant:      pointer.StringPtr(""),
				Owner:       pointer.StringPtr(""),
				Deleted:     util.IntPtr(0),
				IsInEtcd:    util.IntPtr(1),
				Version:     "3",
				GmtCreated:  testTime("2019-02-10T12:27:00Z"),
				GmtFinished: testTimePtr("2019-02-11T12:27:00Z"),
				Resources:   `{"Worker":{"resources":{"requests":{"cpu":"1","memory":"1Gi"}},"replicas":1}}`,
			},
		}, {
			name: "xgboostjob with region",
			args: args{
				job: &v1alpha1.XGBoostJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "xgboostjob-test",
						Namespace:         testNamespace,
						UID:               "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
						ResourceVersion:   "3",
						CreationTimestamp: metav1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					},
					Status: v1alpha1.XGBoostJobStatus{
						JobStatus: apiv1.JobStatus{
							CompletionTime: &metav1.Time{Time: testTime("2019-02-11T12:27:00Z")},
							Conditions:     []apiv1.JobCondition{{Type: apiv1.JobSucceeded}},
						},
					},
					Spec: v1alpha1.XGBoostJobSpec{
						XGBReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
							"Worker": {
								Replicas: pointer.Int32Ptr(1),
								Template: v1.PodTemplateSpec{
									Spec: v1.PodSpec{
										Containers: []v1.Container{
											{
												Name:  testMainContainerName,
												Image: testImage,
												Resources: v1.ResourceRequirements{
													Requests: v1.ResourceList{"cpu": resource.MustParse("1"), "memory": resource.MustParse("1Gi")},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &dmo.Job{
				Name:        "xgboostjob-test",
				JobID:       "6f06d2fd-22c6-11e9-96bb-0242ac1d5327",
				Kind:        v1alpha1.Kind,
				Status:      apiv1.JobSucceeded,
				Namespace:   testNamespace,
				Tenant:      pointer.StringPtr(""),
				Owner:       pointer.StringPtr(""),
				Deleted:     util.IntPtr(0),
				IsInEtcd:    util.IntPtr(1),
				Version:     "3",
				GmtCreated:  testTime("2019-02-10T12:27:00Z"),
				GmtFinished: testTimePtr("2019-02-11T12:27:00Z"),
				Resources:   `{"Worker":{"resources":{"requests":{"cpu":"1","memory":"1Gi"}},"replicas":1}}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, spec, status, err := ExtractTypedJobInfos(tt.args.job)
			if err != nil {
				t.Errorf("failed to extract job info, err: %v", err)
				return
			}
			got, err := ConvertJobToDMOJob(tt.args.job, kind, spec, &status, tt.args.region)
			if err != nil {
				t.Errorf("failed to convert to dmo job, err: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertPodToDMO(): got = %v, want %v", debugJson(got), debugJson(tt.want))
			}
		})
	}
}
