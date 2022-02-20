package istio_less_ingress_controller

import (
	"context"
	"net/http"
	"sync"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func init() {
	caddy.RegisterModule(TrafficControl{})
	httpcaddyfile.RegisterHandlerDirective("traffic_control", parseCaddyfile)
}

type TrafficControl struct {
	cache        Cache
	once         sync.Once
	proxyMap     sync.Map
	caddyContext caddy.Context
	cleanupFunc  func()
}

// CaddyModule returns the Caddy module information.
func (tc TrafficControl) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.traffic_control",
		New: func() caddy.Module {
			return new(TrafficControl)
		},
	}
}

func (tc *TrafficControl) Provision(ctx caddy.Context) error {
	klog.Infof("start to provision...")
	tc.caddyContext = ctx
	if tc.cache == nil {
		tc.cache, tc.cleanupFunc = tc.provision()
	}

	return nil
}

func (tc *TrafficControl) Cleanup() error {
	klog.Infof("start to cleanup...")
	if tc.cleanupFunc != nil {
		tc.cleanupFunc()
	}

	return nil
}

func (tc TrafficControl) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	proxy, ok := tc.getProxy(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
		return nil
	}

	return proxy.ServeHTTP(w, r, next)
}

func (tc *TrafficControl) getProxy(r *http.Request) (caddyhttp.MiddlewareHandler, bool) {
	_, _, dst, match := tc.cache.MatchRequest(r)
	if !match {
		klog.Errorf("match request from cache failed")
		return nil, false
	}

	klog.Infof("request match to dst %v", dst)

	if proxy, ok := tc.proxyMap.Load(dst); ok {
		return proxy.(caddyhttp.MiddlewareHandler), true
	}

	proxy := tc.buildHandler(dst)
	if provisioner, ok := proxy.(caddy.Provisioner); ok {
		provisioner.Provision(tc.caddyContext)
	}

	proxyInMap, loaded := tc.proxyMap.LoadOrStore(dst, proxy)
	if loaded {
		if cleaner, ok := proxy.(caddy.CleanerUpper); ok {
			cleaner.Cleanup()
		}
	}

	return proxyInMap.(caddyhttp.MiddlewareHandler), true
}

func (tc *TrafficControl) buildHandler(addr string) caddyhttp.MiddlewareHandler {
	return &reverseproxy.Handler{
		Upstreams: reverseproxy.UpstreamPool{&reverseproxy.Upstream{Dial: addr}},
	}
}

func (tc *TrafficControl) provision() (Cache, func()) {
	ingressCache := NewIngressCache()
	ctx, stop := context.WithCancel(context.Background())
	tc.once.Do(func() {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			klog.Fatal(err)
		}

		kubeClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			klog.Fatalln(err)
		}

		ingressCache.StartAutoUpdate(ctx, kubeClient)
	})

	return ingressCache, func() { stop() }
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var tc TrafficControl
	return tc, nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*TrafficControl)(nil)
	_ caddyhttp.MiddlewareHandler = (*TrafficControl)(nil)
	_ caddy.CleanerUpper          = (*TrafficControl)(nil)
)
