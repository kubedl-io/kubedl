package storage

import (
	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

var StorageProviders = make(map[string]StorageProvider)

func init() {
	StorageProviders["NFS"] = NewNFSProvider()
	StorageProviders["LocalStorage"] = NewLocalStorageProvider()
	StorageProviders["AWSEfs"] = NewAWSEfsProvider()
}

type StorageProvider interface {
	// CreatePersistentVolume creates the PV for the model
	CreatePersistentVolume(model *modelv1alpha1.Storage, pvName string) *v1.PersistentVolume

	// GetModelPath returns the model path
	GetModelPath(model *modelv1alpha1.Storage) string
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
