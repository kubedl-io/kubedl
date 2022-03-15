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

package serving

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/alibaba/kubedl/apis/model/v1alpha1"
)

func (ir *InferenceReconciler) buildModelLoaderInitContainer(mv *v1alpha1.ModelVersion, sharedVolumeName, destModelPath string) (c *v1.Container, err error) {
	c = &v1.Container{Name: "kubedl-model-loader"}
	c.Image = mv.Status.Image
	c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{Name: sharedVolumeName, MountPath: destModelPath})
	// Model loader is an init container to move model artifacts from source path to destination.
	c.Command = []string{"/bin/sh", "-c", fmt.Sprintf("cp -r %s %s", v1alpha1.DefaultModelPathInImage+"/*", destModelPath)}
	return c, nil
}
