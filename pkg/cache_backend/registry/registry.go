package registry

import (
	"errors"
	"sync"

	"github.com/alibaba/kubedl/apis/cache/v1alpha1"
	"github.com/alibaba/kubedl/pkg/cache_backend"
	"github.com/alibaba/kubedl/pkg/cache_backend/fluid"

	"k8s.io/klog"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var (
	NewCacheEngines []cache_backend.NewCacheEngine
	defaultRegistry = Registry{registry: make(map[string]cache_backend.CacheEngine)}
)

func RegisterCacheBackends(mgr controllerruntime.Manager) {
	for _, newer := range NewCacheEngines {
		cacheEngine := newer(mgr)
		klog.Infof("register cache backend %s", cacheEngine.Name())
		defaultRegistry.Add(cacheEngine)
	}
}

func Get(name string) cache_backend.CacheEngine {
	return defaultRegistry.Get(name)
}

func (r *Registry) Add(cacheEngine cache_backend.CacheEngine) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.registry[cacheEngine.Name()] = cacheEngine
}

func (r *Registry) Get(name string) cache_backend.CacheEngine {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.registry[name]
}

type Registry struct {
	lock     sync.Mutex
	registry map[string]cache_backend.CacheEngine
}

func CacheBackendName(cacheEngine *v1alpha1.CacheEngine) (string, error) {
	switch {
	case cacheEngine.Fluid != nil:
		cache := fluid.Cache{}
		return cache.Name(), nil
	}
	return "", errors.New("NotFound")
}
