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
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"github.com/alibaba/kubedl/pkg/storage/dmo"
)

// ConvertEventToDMOEvent converts a native event object to dmo event.
func ConvertEventToDMOEvent(event v1.Event, region string) (*dmo.Event, error) {
	klog.V(5).Infof("[ConvertEventToDMOEvent] event: %s, type: %s", event.Name, event.Type)
	dmoEvent := &dmo.Event{
		Name:           event.Name,
		Kind:           event.InvolvedObject.Kind,
		Type:           event.Type,
		Reason:         event.Reason,
		Message:        event.Message,
		Count:          event.Count,
		FirstTimestamp: event.FirstTimestamp.Time,
		LastTimestamp:  event.LastTimestamp.Time,
	}
	if region != "" {
		dmoEvent.Region = &region
	}
	return dmoEvent, nil
}
