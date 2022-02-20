/*
Copyright 2020 The KubeDL Authors.
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

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	controllers "github.com/alibaba/kubedl/controllers/model"
)

func init() {
	SetupWithManagerMap[&modelv1alpha1.ModelVersion{}] = func(mgr ctrl.Manager, config options.JobControllerConfiguration) error {
		return controllers.NewModelVersionController(mgr, config).SetupWithManager(mgr)
	}
}
