package storage

import (
	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AliCloudNasProvider struct {
}

func (a *AliCloudNasProvider) GetModelPath(model *modelv1alpha1.Storage) string {
	return model.AlibabaCloudNas.Path
}

func (a *AliCloudNasProvider) CreatePersistentVolume(storage *modelv1alpha1.Storage, pvName string) *corev1.PersistentVolume {

	// Get the volume attributes
	attributes := make(map[string]string, len(storage.AlibabaCloudNas.Attributes))
	for key, val := range storage.AlibabaCloudNas.Attributes {
		attributes[key] = val
	}

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:           "nasplugin.csi.alibabacloud.com",
					VolumeHandle:     pvName,
					VolumeAttributes: attributes,
				},
			},
		},
	}
	return pv
}

func NewAliCloudNasProvider() StorageProvider {
	return &AliCloudNasProvider{}
}
