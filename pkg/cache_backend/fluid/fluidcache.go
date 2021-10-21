package fluid

import (
	"context"

	"github.com/alibaba/kubedl/apis"
	cachev1alpha1 "github.com/alibaba/kubedl/apis/cache/v1alpha1"
	"github.com/alibaba/kubedl/pkg/cache_backend"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Add to runtime scheme so that reflector of go-client will identify this CRD
	// controlled by scheduler.
	apis.AddToSchemes = append(apis.AddToSchemes, fluidv1alpha1.AddToScheme)
}

func NewFluidCache(mgr controllerruntime.Manager) cache_backend.CacheEngine {
	return &Cache{client: mgr.GetClient()}
}

type Cache struct {
	client client.Client
}

func (fluidCache *Cache) CreateCacheJob(cacheBackend *cachev1alpha1.CacheBackend) error {
	// The name of dataset and alluxio runtime must be equal
	dataset := &fluidv1alpha1.Dataset{}
	dataset.Name = cacheBackend.Name
	dataset.Namespace = cacheBackend.Namespace

	alluxioRuntime := &fluidv1alpha1.AlluxioRuntime{}
	alluxioRuntime.Name = cacheBackend.Name
	alluxioRuntime.Namespace = cacheBackend.Namespace

	// Check if dataset has been created, otherwise, create it
	err := fluidCache.client.Get(context.Background(), types.NamespacedName{Namespace: dataset.Namespace, Name: dataset.Name}, dataset)
	if err != nil {
		if errors.IsNotFound(err) {
			err = fluidCache.createDataset(cacheBackend.Spec.CacheEngine.Fluid.Dataset, dataset.Name, dataset.Namespace)
			if err != nil {
				klog.Errorf("failed to create fluid dataset, err: %v", err)
				return err
			}
		} else {
			klog.Errorf("failed to get fluid dataset, err: %v", err)
			return err
		}
	}

	// Check if alluxioruntime has been created
	err = fluidCache.client.Get(context.Background(), types.NamespacedName{Namespace: alluxioRuntime.Namespace, Name: alluxioRuntime.Name}, alluxioRuntime)
	if err != nil {
		if errors.IsNotFound(err) {
			err = fluidCache.createAlluxioRuntime(cacheBackend.Spec.CacheEngine.Fluid.AlluxioRuntime, alluxioRuntime.Name, alluxioRuntime.Namespace)
			if err != nil {
				klog.Errorf("failed to create fluid alluxio runtime, err: %v", err)
				return err
			}
		} else {
			klog.Errorf("failed to get fluid alluxio runtime, err: %v", err)
			return err
		}
	}

	return nil
}

func (fluidCache *Cache) createDataset(dataset *cachev1alpha1.Dataset, name string, namespace string) error {

	var mount []fluidv1alpha1.Mount
	for _, datasource := range dataset.DataSources {
		mount = append(mount, fluidv1alpha1.Mount{
			MountPoint: datasource.Path,
			Name:       datasource.SubdirectoryName,
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

func (fluidCache *Cache) createAlluxioRuntime(alluxioRuntime *cachev1alpha1.AlluxioRuntime, name string, namespace string) (error error) {
	var levels []fluidv1alpha1.Level
	for _, level := range alluxioRuntime.TieredStorage {
		quota := resource.MustParse(level.Quota)
		levels = append(levels, fluidv1alpha1.Level{
			MediumType: common.MediumType(level.MediumType),
			Path:       level.CachePath,
			Quota:      &quota,
		})
	}

	ar := &fluidv1alpha1.AlluxioRuntime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: fluidv1alpha1.AlluxioRuntimeSpec{
			Replicas: alluxioRuntime.Replicas,
			TieredStore: fluidv1alpha1.TieredStore{
				Levels: levels,
			},
		},
	}

	err := fluidCache.client.Create(context.Background(), ar)

	return err
}

func (fluidCache *Cache) Name() string {
	return "fluid"
}
