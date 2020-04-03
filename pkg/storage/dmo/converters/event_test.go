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
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"reflect"
	"testing"
)

const (
	testReason  = "reason for test event"
	testMessage = "message for test event"
)

func TestConvertEventToDMOEvent(t *testing.T) {
	type args struct {
		e      corev1.Event
		region string
	}
	tests := []struct {
		name string
		args *args
		want *dmo.Event
	}{
		{
			name: "normal event without region",
			args: &args{
				e: corev1.Event{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-event",
						Namespace: testNamespace,
					},
					InvolvedObject: corev1.ObjectReference{
						Name:      "test-tfjob",
						Namespace: testNamespace,
						Kind:      "TFJob",
					},
					Reason:         testReason,
					Message:        testMessage,
					FirstTimestamp: v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					LastTimestamp:  v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					Type:           corev1.EventTypeNormal,
				},
				region: "",
			},
			want: &dmo.Event{
				Name:           "test-event",
				Kind:           "TFJob",
				Type:           corev1.EventTypeNormal,
				Reason:         testReason,
				Message:        testMessage,
				Count:          0,
				FirstTimestamp: testTime("2019-02-10T12:27:00Z"),
				LastTimestamp:  testTime("2019-02-10T12:27:00Z"),
			},
		},
		{
			name: "normal event with region",
			args: &args{
				e: corev1.Event{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-event",
						Namespace: testNamespace,
					},
					InvolvedObject: corev1.ObjectReference{
						Name:      "test-tfjob",
						Namespace: testNamespace,
						Kind:      "TFJob",
					},
					Reason:         testReason,
					Message:        testMessage,
					FirstTimestamp: v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					LastTimestamp:  v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					Type:           corev1.EventTypeNormal,
				},
				region: testRegion,
			},
			want: &dmo.Event{
				Name:           "test-event",
				Kind:           "TFJob",
				Type:           corev1.EventTypeNormal,
				Reason:         testReason,
				Message:        testMessage,
				Region:         pointer.StringPtr(testRegion),
				Count:          0,
				FirstTimestamp: testTime("2019-02-10T12:27:00Z"),
				LastTimestamp:  testTime("2019-02-10T12:27:00Z"),
			},
		},
		{
			name: "warning event with region",
			args: &args{
				e: corev1.Event{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-event",
						Namespace: testNamespace,
					},
					InvolvedObject: corev1.ObjectReference{
						Name:      "test-tfjob",
						Namespace: testNamespace,
						Kind:      "TFJob",
					},
					Reason:         testReason,
					Message:        testMessage,
					FirstTimestamp: v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					LastTimestamp:  v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					Type:           corev1.EventTypeWarning,
				},
				region: testRegion,
			},
			want: &dmo.Event{
				Name:           "test-event",
				Kind:           "TFJob",
				Type:           corev1.EventTypeWarning,
				Reason:         testReason,
				Message:        testMessage,
				Region:         pointer.StringPtr(testRegion),
				Count:          0,
				FirstTimestamp: testTime("2019-02-10T12:27:00Z"),
				LastTimestamp:  testTime("2019-02-10T12:27:00Z"),
			},
		},
		{
			name: "normal event with counts",
			args: &args{
				e: corev1.Event{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-event",
						Namespace: testNamespace,
					},
					InvolvedObject: corev1.ObjectReference{
						Name:      "test-tfjob",
						Namespace: testNamespace,
						Kind:      "TFJob",
					},
					Reason:         testReason,
					Message:        testMessage,
					FirstTimestamp: v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					LastTimestamp:  v1.Time{Time: testTime("2019-02-10T12:27:00Z")},
					Type:           corev1.EventTypeNormal,
					Count:          10,
				},
				region: testRegion,
			},
			want: &dmo.Event{
				Name:           "test-event",
				Kind:           "TFJob",
				Type:           corev1.EventTypeNormal,
				Reason:         testReason,
				Message:        testMessage,
				Region:         pointer.StringPtr(testRegion),
				Count:          10,
				FirstTimestamp: testTime("2019-02-10T12:27:00Z"),
				LastTimestamp:  testTime("2019-02-10T12:27:00Z"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertEventToDMOEvent(tt.args.e, tt.args.region)
			if err != nil {
				t.Errorf("failed to convert to dmo event, err: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertEventToDMOEvent(): got = %v, want %v", debugJson(got), debugJson(tt.want))
			}
		})
	}
}
