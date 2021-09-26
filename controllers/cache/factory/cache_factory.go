package cachefactory

import (
	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

var CacheFactory = make(map[string]CacheProvider)

func init() {
	CacheFactory["fluid"] = NewFluidCache()

}

type CacheProvider interface {
	CreateCacheJob(cacheBackend *cachev1alpha1.CacheBackend, pvcName string) *v1.PersistentVolumeClaim
}

func GetCacheProvider(cacheEngine *cachev1alpha1.CacheEngine) CacheProvider {
	if cacheEngine.Fluid != nil {
		return CacheFactory["fluid"]
	}
	return nil
}
