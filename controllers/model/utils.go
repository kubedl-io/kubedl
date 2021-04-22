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

package controllers

func GetJobModelVersionName(postfix string) string {
	return "model-version-" + postfix
}

func GetModelVersionPVName(modelName string) string {
	return "model-version-pv-" + modelName
}

func GetModelVersionPVClaimName(modelName string) string {
	return "model-version-pvc-" + modelName
}

func GetModelVersionPVNameByNode(modelName string, nodeName string) string {
	return "model-version-pv-" + modelName + "-" + nodeName
}

func GetModelVersionPVClaimNameByNode(modelName string, nodeName string) string {
	return "model-version-pvc-" + modelName + "-" + nodeName
}

func GetBuildImagePodName(modelName string, versionId string) string {
	return "image-build-" + modelName + "-v" + versionId
}
