package cache_backend

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
)

type CacheEngine interface {
	CreateCacheJob(cacheBackend *cachev1alpha1.CacheBackend) (error error)

	Name() string
}

type NewCacheEngine func(client client.Client) CacheEngine
