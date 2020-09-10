/*
Copyright 2020 The Alibaba Authors.

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
	"github.com/alibaba/kubedl/api/marsjob/v1alpha1"
	controllers "github.com/alibaba/kubedl/controllers/mars"

	"github.com/alibaba/kubedl/pkg/job_controller"
	"sigs.k8s.io/controller-runtime"
)

func init() {
	SetupWithManagerMap[&v1alpha1.MarsJob{}] = func(mgr controllerruntime.Manager, config job_controller.JobControllerConfiguration) error {
		return controllers.NewReconciler(mgr, config).SetupWithManager(mgr)
	}
}
