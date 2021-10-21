package cache_backend

import (
	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"

	controllerruntime "sigs.k8s.io/controller-runtime"
)

type CacheEngine interface {
	CreateCacheJob(cacheBackend *cachev1alpha1.CacheBackend) (error error)

	Name() string
}

type NewCacheEngine func(mgr controllerruntime.Manager) CacheEngine
