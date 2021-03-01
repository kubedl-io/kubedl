package storage

import (
	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

var StorageProviders = make(map[string]StorageProvider)

func init() {
	StorageProviders["AliCloudNas"] = NewAliCloudNasProvider()
	StorageProviders["LocalStorage"] = NewLocalStorageProvider()

}

type StorageProvider interface {
	CreatePersistentVolume(model *modelv1alpha1.Storage, pvName string) *v1.PersistentVolume
}

func GetStorageProvider(storage *modelv1alpha1.Storage) StorageProvider {
	if storage.AliCloudNas != nil {
		return StorageProviders["AliCloudNas"]
	}
	if storage.LocalStorage != nil {
		return StorageProviders["LocalStorage"]
	}
	return nil
}
