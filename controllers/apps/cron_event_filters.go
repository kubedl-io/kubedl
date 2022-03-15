/*
Copyright 2021 The Alibaba Authors.

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

package cron

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/alibaba/kubedl/apis/apps/v1alpha1"
)

// Event filter functions defined down below filter those workloads whom controlled
// by CronTask.

func onCronWorkloadCreate(evt event.CreateEvent) bool {
	return isControlledByCronTask(evt.Object)
}

func onCronWorkloadUpdate(newEvt event.UpdateEvent) bool {
	return isControlledByCronTask(newEvt.ObjectNew)
}

func onCronWorkloadDelete(evt event.DeleteEvent) bool {
	return isControlledByCronTask(evt.Object)
}

func isControlledByCronTask(obj metav1.Object) bool {
	for i := range obj.GetOwnerReferences() {
		ref := &obj.GetOwnerReferences()[i]
		if ref.Kind == v1alpha1.KindCron && ref.Controller != nil && *ref.Controller {
			return true
		}
	}
	return false
}
