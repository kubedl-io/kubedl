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

// Package tensorflow provides a Kubernetes controller for a TFJob resource.
package tensorflow

import (
	tfv1 "github.com/alibaba/kubedl/api/tensorflow/v1"
)

// ContainChieforMasterSpec returns true if the tfjob contains chief or master spec.
func ContainChieforMasterSpec(tfJob *tfv1.TFJob) bool {
	if _, ok := tfJob.Spec.TFReplicaSpecs[tfv1.TFReplicaTypeChief]; ok {
		return true
	} else if _, ok := tfJob.Spec.TFReplicaSpecs[tfv1.TFReplicaTypeMaster]; ok {
		return true
	}
	return false
}
