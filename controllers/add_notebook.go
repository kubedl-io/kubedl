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

	notebookv1alpha1 "github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	controllers "github.com/alibaba/kubedl/controllers/notebook"
)

func init() {
	SetupWithManagerMap[&notebookv1alpha1.Notebook{}] = func(mgr ctrl.Manager, config options.JobControllerConfiguration) error {
		return controllers.NewNotebookController(mgr, config).SetupWithManager(mgr)
	}
}
