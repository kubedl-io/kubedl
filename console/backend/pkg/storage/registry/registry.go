package registry

import (
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/backends/registry"
	"k8s.io/klog"
)

var (
	newObjectBackends      []func() backends.ObjectStorageBackend
	newEventBackends       []func() backends.EventStorageBackend
	defaultBackendRegistry = registry.NewBackendRegistry()
)

func RegisterStorageBackends() {
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
