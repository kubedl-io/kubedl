package cachefactory

import (
	"context"
	"strconv"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
)

type FluidCache struct {
	client client.Client
}

func (fluidCache *FluidCache) CreateCacheJob(cacheBackend *cachev1alpha1.CacheBackend, pvcName string) (error error) {

	// the name of dataset and alluxioruntime must be equal
	dataset := &fluidv1alpha1.Dataset{}
	datasetName := pvcName
	datasetNameSpace := cacheBackend.Namespace

	alluxioRuntime := &fluidv1alpha1.AlluxioRuntime{}
	alluxioRuntimeName := pvcName
	alluxioRuntimeNameSpace := cacheBackend.Namespace

	err := fluidCache.client.Get(context.Background(), types.NamespacedName{Namespace: datasetNameSpace, Name: datasetName}, dataset)
	if err != nil {
		if errors.IsNotFound(err) {
			err = fluidCache.createDataset(cacheBackend.Spec.CacheEngine.Fluid.Dataset, datasetName, datasetNameSpace)
			if err != nil {
				klog.Errorf("failed to create fluid dataset")
				return err
			}
		} else {
			klog.Errorf("failed to get fluid dataset")
			return err
		}
	}

	err = fluidCache.client.Get(context.Background(), types.NamespacedName{Namespace: alluxioRuntimeNameSpace, Name: alluxioRuntimeName}, alluxioRuntime)
	if err != nil {
		if errors.IsNotFound(err) {
			err = fluidCache.createAlluxioRuntime(cacheBackend.Spec.CacheEngine.Fluid.AlluxioRuntime, alluxioRuntimeName, alluxioRuntimeNameSpace)
			if err != nil {
				klog.Errorf("failed to create fluid alluxio runtime")
				return err
			}
		} else {
			klog.Errorf("failed to get fluid alluxio runtime")
			return err
		}
	}

	return nil
}

func (fluidCache *FluidCache) createDataset(dataset *cachev1alpha1.Dataset, name string, namespace string) (error error) {

	mount := []fluidv1alpha1.Mount{}
	for i, m := range dataset.Mounts {
		mount = append(mount, fluidv1alpha1.Mount{
			MountPoint: m.DataSource,
			Name:       name + "-" + strconv.Itoa(i),
		})
	}

	ds := &fluidv1alpha1.Dataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: fluidv1alpha1.DatasetSpec{
			Mounts: mount,
		},
	}

	err := fluidCache.client.Create(context.Background(), ds)

	return err
}

func (fluidCache *FluidCache) createAlluxioRuntime(alluxioRuntime *cachev1alpha1.AlluxioRuntime, name string, namespace string) (error error) {
	levels := []fluidv1alpha1.Level{}
	for _, l := range alluxioRuntime.TierdStorage {
		quota := resource.MustParse(l.Quota)
		levels = append(levels, fluidv1alpha1.Level{
			MediumType: common.MediumType(l.MediumType),
			Path:       l.CachePath,
			Quota:      &quota,
		})
	}

	ar := &fluidv1alpha1.AlluxioRuntime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: fluidv1alpha1.AlluxioRuntimeSpec{
			TieredStore: fluidv1alpha1.TieredStore{
				Levels: levels,
			},
		},
	}

	err := fluidCache.client.Create(context.Background(), ar)

	return err
}

func NewFluidCache() CacheProvider {
	return &FluidCache{}
}
