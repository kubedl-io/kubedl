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

package runtime

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func NewRawExtensionCodec(scheme *runtime.Scheme) *RawExtensionCodec {
	return &RawExtensionCodec{scheme: scheme, codec: serializer.NewCodecFactory(scheme)}
}

type RawExtensionCodec struct {
	scheme *runtime.Scheme
	codec  serializer.CodecFactory
}

func (rc *RawExtensionCodec) DecodeRaw(rawObj runtime.RawExtension, into runtime.Object) error {
	// we error out if rawObj is an empty object.
	if len(rawObj.Raw) == 0 {
		return fmt.Errorf("there is no content to decode")
	}

	deserializer := rc.codec.UniversalDeserializer()
	return runtime.DecodeInto(deserializer, rawObj.Raw, into)
}

func (rc *RawExtensionCodec) EncodeRaw(obj runtime.Object) (*runtime.RawExtension, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return &runtime.RawExtension{Raw: data, Object: obj}, nil
}

// FailedPodContents collects failed reasons while with its exit codes of failed pods.
// key is {reason}-{exit code}, value is a slice of related pods.
type FailedPodContents map[string][]string

func (fc FailedPodContents) Add(pod *v1.Pod, exitCode int32) {
	key := fmt.Sprintf("%s-%d", pod.Status.Reason, exitCode)
	if len(fc[key]) > 10 {
		return
	}
	fc[key] = append(fc[key], pod.Name)
}

func (fc FailedPodContents) String() string {
	if fc == nil {
		return ""
	}
	bytes, _ := json.Marshal(&fc)
	return string(bytes)
}
