// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

const (
	// GroupName is the group name use in this package.
	GroupName = "kubeflow.org"
	// Kind is the kind name.
	Kind = "PyTorchJob"
	// GroupVersion is the version.
	GroupVersion = "v1"
	// Plural is the Plural for pytorchJob.
	Plural = "pytorchjobs"
	// Singular is the singular for pytorchJob.
	Singular = "pytorchjob"
	// PytorchCRD is the CRD name for PytorchJob.
	PytorchCRD = "pytorchjobs.kubeflow.org"
)

var (
	// SchemeGroupVersion is the group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	// SchemeGroupVersionKind is the GroupVersionKind of the resource.
	SchemeGroupVersionKind = SchemeGroupVersion.WithKind(Kind)
)

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	SchemeBuilder.SchemeBuilder.Register(addDefaultingFuncs)
}

// Resource takes an unqualified resource and returns a Group-qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
