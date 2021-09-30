package cachefactory

import (
	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
)

var CacheFactory = make(map[string]CacheProvider)

func init() {
	CacheFactory["fluid"] = NewFluidCache()

}

type CacheProvider interface {
	CreateCacheJob(cacheBackend *cachev1alpha1.CacheBackend, pvcName string) (error error)
}

func GetCacheProvider(cacheEngine *cachev1alpha1.CacheEngine) CacheProvider {
	if cacheEngine.Fluid != nil {
		return CacheFactory["fluid"]
	}
	return nil
}
