package client

import (
	"context"

	"github.com/alibaba/kubedl/apis"

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

	cmgr.ctrlClient, err = client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader: ctrlCache,
		Client:      c,
	})
	if err != nil {
		klog.Fatal(err)
	}
}

func Start(ctx context.Context) {
	go func() {
		_ = cmgr.ctrlCache.Start(ctx)
	}()
}

func IndexField(obj client.Object, field string, extractValue client.IndexerFunc) error {
	return cmgr.ctrlCache.IndexField(context.Background(), obj, field, extractValue)
}

func GetCtrlClient() client.Client {
	return cmgr.ctrlClient
}

func GetScheme() *runtime.Scheme {
	return cmgr.scheme
}
