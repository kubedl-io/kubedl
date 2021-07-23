package storage

import (
	"github.com/alibaba/kubedl/console/backend/pkg/storage/apiserver"
	"github.com/alibaba/kubedl/console/backend/pkg/storage/proxy"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"k8s.io/klog"
)

var defaultRegistry map[string]backends.ObjectStorageBackend

func RegisterObjectBackend() {
	defaultRegistry = make(map[string]backends.ObjectStorageBackend)
	objects := []backends.ObjectStorageBackend{
		apiserver.NewAPIServerBackendService(),
		proxy.NewProxyBackendService(),
	}
	for _, object := range objects {
		klog.Infof("register new object backend: %s", object.Name())
		AddObjectBackend(object)
	}
}

func GetObjectBackend(name string) backends.ObjectStorageBackend {
	return defaultRegistry[name]
}

func AddObjectBackend(object backends.ObjectStorageBackend) {
	defaultRegistry[object.Name()] = object
}
