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

package gang_schedule

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AppendOwnerReference(obj metav1.Object, newReference metav1.OwnerReference) {
	ownerReferences := obj.GetOwnerReferences()
	for _, reference := range ownerReferences {
		if ownerReferenceEquals(reference, newReference) {
			return
		}
	}
	ownerReferences = append(ownerReferences, newReference)
	obj.SetOwnerReferences(ownerReferences)
}

func ownerReferenceEquals(ref1, ref2 metav1.OwnerReference) bool {
	if ref1.APIVersion != ref2.APIVersion ||
		ref1.Kind != ref2.Kind ||
		ref1.UID != ref2.UID {
		return false
	}
	return true
}
