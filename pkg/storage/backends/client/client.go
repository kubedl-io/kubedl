package client

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClientManager is a set of abstract methods to get client to connect apiserver
type ClientManager interface {
	// GetCtrlClient returns a client configured with the Config. This client may
	// not be a fully "direct" client -- it may read from a cache, for
	// instance.
	GetCtrlClient() client.Client

	// GetScheme returns an initialized Scheme
	GetScheme() *runtime.Scheme

	// IndexField adds an index with the given field name on the given object type
	// by using the given function to extract the value for that field.
	IndexField(obj runtime.Object, field string, extractValue client.IndexerFunc) error
}

var clientMgr ClientManager

func InstallClientManager(mgr ClientManager) {
	clientMgr = mgr
}

func GetCtrlClient() client.Client {
	if clientMgr == nil {
		klog.Fatal("get clientMgr fail, clientMgr is nil")
	}
	return clientMgr.GetCtrlClient()
}

func GetScheme() *runtime.Scheme {
	if clientMgr == nil {
		klog.Fatal("get clientMgr fail, clientMgr is nil")
	}
	return clientMgr.GetScheme()
}

func IndexField(obj runtime.Object, field string, extractValue client.IndexerFunc) error {
	if clientMgr == nil {
		klog.Fatal("get clientMgr fail, clientMgr is nil")
	}
	return clientMgr.IndexField(obj, field, extractValue)
}

