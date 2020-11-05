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

// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha1 contains API Schema definitions for the xdl v1alpha1 API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/alibaba/kubedl/api/xdl
// +k8s:defaulter-gen=TypeMeta
// +groupName=xdl.kubedl.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

const (
	// GroupName is the group name use in this package.
	GroupName = "xdl.kubedl.io"
	// Kind is the kind name.
	Kind = "XDLJob"
	// GroupVersion is the version.
	GroupVersion = "v1alpha1"
	// Plural is the Plural for XDLJob.
	Plural = "xdljobs"
	// Singular is the singular for XDLJob.
	Singular = "xdljob"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

	// SchemeGroupVersionKind is the GroupVersionKind of the resource.
	SchemeGroupVersionKind = SchemeGroupVersion.WithKind(Kind)

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

// Resource is required by pkg/client/listers/...
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	SchemeBuilder.SchemeBuilder.Register(addDefaultingFuncs)
}
