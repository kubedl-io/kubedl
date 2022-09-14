package test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/kubedl/apis/cache/v1alpha1"
)

func NewFluidCacheBackend(name string, namespace string) *v1alpha1.CacheBackend {
	cacheBackend := &v1alpha1.CacheBackend{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.CacheBackendSpec{
			CacheBackendName: name,
			MountPath:        "/test/mount/path",
			Dataset:          &v1alpha1.Dataset{DataSources: []v1alpha1.DataSource{}},
			CacheEngine: &v1alpha1.CacheEngine{
				Fluid: &v1alpha1.Fluid{
					AlluxioRuntime: &v1alpha1.AlluxioRuntime{
						Replicas:      1,
						TieredStorage: []v1alpha1.Level{},
					},
				},
			},
			Options: v1alpha1.Options{IdleTime: 60},
		},
		Status: v1alpha1.CacheBackendStatus{
			CacheEngine: "fluid",
			CacheStatus: v1alpha1.CacheCreating,
		},
	}

	cacheBackend.Spec.Dataset.DataSources = append(cacheBackend.Spec.Dataset.DataSources, v1alpha1.DataSource{
		Location:   "/test/dataset/location",
		SubDirName: "/test/sub/directory/name",
	})

	cacheBackend.Spec.CacheEngine.Fluid.AlluxioRuntime.TieredStorage =
		append(cacheBackend.Spec.CacheEngine.Fluid.AlluxioRuntime.TieredStorage, v1alpha1.Level{
			CachePath:  "/test/path",
			Quota:      "1Gi",
			MediumType: "MEM",
		})

	return cacheBackend
}
