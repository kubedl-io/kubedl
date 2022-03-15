package storage

import (
	v1 "k8s.io/api/core/v1"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
)

var StorageProviders = make(map[string]StorageProvider)

func init() {
	StorageProviders["NFS"] = NewNFSProvider()
	StorageProviders["LocalStorage"] = NewLocalStorageProvider()
	StorageProviders["AWSEfs"] = NewAWSEfsProvider()
}

type StorageProvider interface {
	// CreatePersistentVolume creates the PV for the model
	CreatePersistentVolume(mv *modelv1alpha1.Storage, pvName string) *v1.PersistentVolume

	// Add the model volume and mountpath to the pod spec
	AddModelVolumeToPodSpec(mv *modelv1alpha1.Storage, pod *v1.PodTemplateSpec)

	// Get the model mount path inside the container
	GetModelMountPath(mv *modelv1alpha1.Storage) string
}

func GetStorageProvider(storage *modelv1alpha1.Storage) StorageProvider {
	if storage.NFS != nil {
		return StorageProviders["NFS"]
	}
	if storage.LocalStorage != nil {
		return StorageProviders["LocalStorage"]
	}
	if storage.AWSEfs != nil {
		return StorageProviders["AWSEfs"]
	}
	return nil
}
