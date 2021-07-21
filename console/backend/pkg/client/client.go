package client

import (
	"context"

	"github.com/alibaba/kubedl/apis"
	clientmgr "github.com/alibaba/kubedl/pkg/storage/backends/client"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type clientManager struct {
	config     *rest.Config
	scheme     *runtime.Scheme
	ctrlCache  cache.Cache
	ctrlClient client.Client
}

var cmgr = &clientManager{}

func Init() {
	cmgr.config = ctrl.GetConfigOrDie()

	cmgr.scheme = runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(cmgr.scheme)
	_ = apis.AddToScheme(cmgr.scheme)

	ctrlCache, err := cache.New(cmgr.config, cache.Options{Scheme: cmgr.scheme})
	if err != nil {
		klog.Fatal(err)
	}
	cmgr.ctrlCache = ctrlCache

	c, err := client.New(cmgr.config, client.Options{Scheme: cmgr.scheme})
	if err != nil {
		klog.Fatal(err)
	}

	cmgr.ctrlClient = &client.DelegatingClient{
		Reader: &client.DelegatingReader{
			CacheReader:  ctrlCache,
			ClientReader: c,
		},
		Writer:       c,
		StatusClient: c,
	}

	clientmgr.InstallClientManager(cmgr)
}

func Start() {
	go func() {
		stop := make(chan struct{})
		cmgr.ctrlCache.Start(stop)
	}()
}

func (c *clientManager) IndexField(obj runtime.Object, field string, extractValue client.IndexerFunc) error {
	return c.ctrlCache.IndexField(context.Background(), obj, field, extractValue)
}

func (c *clientManager) GetCtrlClient() client.Client {
	return c.ctrlClient
}

func (c *clientManager) GetScheme() *runtime.Scheme {
	return c.scheme
}
