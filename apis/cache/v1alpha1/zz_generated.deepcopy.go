//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AlluxioRuntime) DeepCopyInto(out *AlluxioRuntime) {
	*out = *in
	if in.TieredStorage != nil {
		in, out := &in.TieredStorage, &out.TieredStorage
		*out = make([]Level, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AlluxioRuntime.
func (in *AlluxioRuntime) DeepCopy() *AlluxioRuntime {
	if in == nil {
		return nil
	}
	out := new(AlluxioRuntime)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheBackend) DeepCopyInto(out *CacheBackend) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheBackend.
func (in *CacheBackend) DeepCopy() *CacheBackend {
	if in == nil {
		return nil
	}
	out := new(CacheBackend)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CacheBackend) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheBackendList) DeepCopyInto(out *CacheBackendList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CacheBackend, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheBackendList.
func (in *CacheBackendList) DeepCopy() *CacheBackendList {
	if in == nil {
		return nil
	}
	out := new(CacheBackendList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CacheBackendList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheBackendSpec) DeepCopyInto(out *CacheBackendSpec) {
	*out = *in
	if in.Dataset != nil {
		in, out := &in.Dataset, &out.Dataset
		*out = new(Dataset)
		(*in).DeepCopyInto(*out)
	}
	if in.CacheEngine != nil {
		in, out := &in.CacheEngine, &out.CacheEngine
		*out = new(CacheEngine)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheBackendSpec.
func (in *CacheBackendSpec) DeepCopy() *CacheBackendSpec {
	if in == nil {
		return nil
	}
	out := new(CacheBackendSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheBackendStatus) DeepCopyInto(out *CacheBackendStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheBackendStatus.
func (in *CacheBackendStatus) DeepCopy() *CacheBackendStatus {
	if in == nil {
		return nil
	}
	out := new(CacheBackendStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheEngine) DeepCopyInto(out *CacheEngine) {
	*out = *in
	if in.Fluid != nil {
		in, out := &in.Fluid, &out.Fluid
		*out = new(Fluid)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheEngine.
func (in *CacheEngine) DeepCopy() *CacheEngine {
	if in == nil {
		return nil
	}
	out := new(CacheEngine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataSource) DeepCopyInto(out *DataSource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataSource.
func (in *DataSource) DeepCopy() *DataSource {
	if in == nil {
		return nil
	}
	out := new(DataSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dataset) DeepCopyInto(out *Dataset) {
	*out = *in
	if in.DataSources != nil {
		in, out := &in.DataSources, &out.DataSources
		*out = make([]DataSource, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dataset.
func (in *Dataset) DeepCopy() *Dataset {
	if in == nil {
		return nil
	}
	out := new(Dataset)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Fluid) DeepCopyInto(out *Fluid) {
	*out = *in
	if in.AlluxioRuntime != nil {
		in, out := &in.AlluxioRuntime, &out.AlluxioRuntime
		*out = new(AlluxioRuntime)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Fluid.
func (in *Fluid) DeepCopy() *Fluid {
	if in == nil {
		return nil
	}
	out := new(Fluid)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Level) DeepCopyInto(out *Level) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Level.
func (in *Level) DeepCopy() *Level {
	if in == nil {
		return nil
	}
	out := new(Level)
	in.DeepCopyInto(out)
	return out
}
