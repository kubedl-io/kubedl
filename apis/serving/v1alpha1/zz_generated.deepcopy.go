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
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AutoScaleStrategy) DeepCopyInto(out *AutoScaleStrategy) {
	*out = *in
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	if in.MaxReplicas != nil {
		in, out := &in.MaxReplicas, &out.MaxReplicas
		*out = new(int32)
		**out = **in
	}
	out.AutoScaler = in.AutoScaler
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AutoScaleStrategy.
func (in *AutoScaleStrategy) DeepCopy() *AutoScaleStrategy {
	if in == nil {
		return nil
	}
	out := new(AutoScaleStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BatchingStrategy) DeepCopyInto(out *BatchingStrategy) {
	*out = *in
	if in.TimeoutSeconds != nil {
		in, out := &in.TimeoutSeconds, &out.TimeoutSeconds
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BatchingStrategy.
func (in *BatchingStrategy) DeepCopy() *BatchingStrategy {
	if in == nil {
		return nil
	}
	out := new(BatchingStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Inference) DeepCopyInto(out *Inference) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Inference.
func (in *Inference) DeepCopy() *Inference {
	if in == nil {
		return nil
	}
	out := new(Inference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Inference) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InferenceList) DeepCopyInto(out *InferenceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Inference, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InferenceList.
func (in *InferenceList) DeepCopy() *InferenceList {
	if in == nil {
		return nil
	}
	out := new(InferenceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InferenceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InferenceSpec) DeepCopyInto(out *InferenceSpec) {
	*out = *in
	if in.Predictors != nil {
		in, out := &in.Predictors, &out.Predictors
		*out = make([]PredictorSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InferenceSpec.
func (in *InferenceSpec) DeepCopy() *InferenceSpec {
	if in == nil {
		return nil
	}
	out := new(InferenceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InferenceStatus) DeepCopyInto(out *InferenceStatus) {
	*out = *in
	if in.PredictorStatuses != nil {
		in, out := &in.PredictorStatuses, &out.PredictorStatuses
		*out = make([]PredictorStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InferenceStatus.
func (in *InferenceStatus) DeepCopy() *InferenceStatus {
	if in == nil {
		return nil
	}
	out := new(InferenceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PredictorSpec) DeepCopyInto(out *PredictorSpec) {
	*out = *in
	if in.ModelPath != nil {
		in, out := &in.ModelPath, &out.ModelPath
		*out = new(string)
		**out = **in
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.TrafficWeight != nil {
		in, out := &in.TrafficWeight, &out.TrafficWeight
		*out = new(int32)
		**out = **in
	}
	in.Template.DeepCopyInto(&out.Template)
	if in.AutoScale != nil {
		in, out := &in.AutoScale, &out.AutoScale
		*out = new(AutoScaleStrategy)
		(*in).DeepCopyInto(*out)
	}
	if in.Batching != nil {
		in, out := &in.Batching, &out.Batching
		*out = new(BatchingStrategy)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PredictorSpec.
func (in *PredictorSpec) DeepCopy() *PredictorSpec {
	if in == nil {
		return nil
	}
	out := new(PredictorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PredictorStatus) DeepCopyInto(out *PredictorStatus) {
	*out = *in
	if in.TrafficPercent != nil {
		in, out := &in.TrafficPercent, &out.TrafficPercent
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PredictorStatus.
func (in *PredictorStatus) DeepCopy() *PredictorStatus {
	if in == nil {
		return nil
	}
	out := new(PredictorStatus)
	in.DeepCopyInto(out)
	return out
}
