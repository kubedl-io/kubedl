package controllers

func GetModelPVName(modelName string) string {
	return "model-pv-" + modelName
}

func GetModelPVCName(modelName string) string {
	return "model-pvc-" + modelName
}

func GetModelVersionPVName(modelName string) string {
	return "model-version-pv-" + modelName
}

func GetModelVersionPVCName(modelName string) string {
	return "model-version-pvc-" + modelName
}

func GetBuildImagePodName(modelName string, versionId string) string {
	return "image-build-" + modelName + "-v" + versionId
}
