package storage

import (
	"k8s.io/klog"

	event "github.com/alibaba/kubedl/console/backend/pkg/storage/events/apiserver"
	object "github.com/alibaba/kubedl/console/backend/pkg/storage/objects/apiserver"
	"github.com/alibaba/kubedl/console/backend/pkg/storage/objects/proxy"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/backends/events/aliyun_sls"
	"github.com/alibaba/kubedl/pkg/storage/backends/registry"
)

var defaultBackendRegistry = registry.NewBackendRegistry()

func RegisterStorageBackends() {
	newObjectBackends := []func() backends.ObjectStorageBackend{
		object.NewAPIServerObjectBackend,
		proxy.NewProxyObjectBackend,
	}
	newEventBackends := []func() backends.EventStorageBackend{
		event.NewAPIServerEventBackend,
		aliyun_sls.NewSLSEventBackend,
	}

	for _, newBackend := range newObjectBackends {
		b := newBackend()
		klog.Infof("register new object backend: %s", b.Name())
		AddObjectBackend(b)
	}
	for _, newBackend := range newEventBackends {
		b := newBackend()
		klog.Infof("register new event backend: %s", b.Name())
		AddEventBackend(b)
	}
}

func AddObjectBackend(objBackend backends.ObjectStorageBackend) {
	defaultBackendRegistry.AddObjectBackend(objBackend)
}

func GetObjectBackend(name string) backends.ObjectStorageBackend {
	return defaultBackendRegistry.GetObjectBackend(name)
}

func AddEventBackend(eventBackend backends.EventStorageBackend) {
	defaultBackendRegistry.AddEventBackend(eventBackend)
}

func GetEventBackend(name string) backends.EventStorageBackend {
	return defaultBackendRegistry.GetEventBackend(name)
}
