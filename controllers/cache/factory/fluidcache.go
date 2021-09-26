package cachefactory

import (
	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	v1 "k8s.io/api/core/v1"
	// fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
)

type FluidCache struct {
}

func (fluidCache *FluidCache) CreateCacheJob(cacheEngine *cachev1alpha1.CacheBackend, pvcName string) *v1.PersistentVolumeClaim {

}

func NewFluidCache() CacheProvider {
	return &FluidCache{}
}
