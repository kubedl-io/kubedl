//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2024 The KubeDL Authors.

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
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cron) DeepCopyInto(out *Cron) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cron.
func (in *Cron) DeepCopy() *Cron {
	if in == nil {
		return nil
	}
	out := new(Cron)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Cron) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronHistory) DeepCopyInto(out *CronHistory) {
	*out = *in
	in.Object.DeepCopyInto(&out.Object)
	if in.Created != nil {
		in, out := &in.Created, &out.Created
		*out = (*in).DeepCopy()
	}
	if in.Finished != nil {
		in, out := &in.Finished, &out.Finished
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronHistory.
func (in *CronHistory) DeepCopy() *CronHistory {
	if in == nil {
		return nil
	}
	out := new(CronHistory)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronList) DeepCopyInto(out *CronList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Cron, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronList.
func (in *CronList) DeepCopy() *CronList {
	if in == nil {
		return nil
	}
	out := new(CronList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CronList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronSpec) DeepCopyInto(out *CronSpec) {
	*out = *in
	in.CronPolicy.DeepCopyInto(&out.CronPolicy)
	in.CronTemplate.DeepCopyInto(&out.CronTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronSpec.
func (in *CronSpec) DeepCopy() *CronSpec {
	if in == nil {
		return nil
	}
	out := new(CronSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronStatus) DeepCopyInto(out *CronStatus) {
	*out = *in
	if in.Active != nil {
		in, out := &in.Active, &out.Active
		*out = make([]v1.ObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.History != nil {
		in, out := &in.History, &out.History
		*out = make([]CronHistory, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.LastScheduleTime != nil {
		in, out := &in.LastScheduleTime, &out.LastScheduleTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronStatus.
func (in *CronStatus) DeepCopy() *CronStatus {
	if in == nil {
		return nil
	}
	out := new(CronStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronTemplateSpec) DeepCopyInto(out *CronTemplateSpec) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.Workload != nil {
		in, out := &in.Workload, &out.Workload
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronTemplateSpec.
func (in *CronTemplateSpec) DeepCopy() *CronTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(CronTemplateSpec)
	in.DeepCopyInto(out)
	return out
}
