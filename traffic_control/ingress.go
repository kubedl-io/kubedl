package istio_less_ingress_controller

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

type ingressEntry struct {
	ingress *v1beta1.Ingress
	path    v1beta1.HTTPIngressPath
}

type ingressCache struct {
	mutex sync.Mutex

	hostToIngressEntry map[string][]ingressEntry
	ingressMap         map[string]*v1beta1.Ingress // ingressKey(ns/name) => ingress
	reloadChannel      chan struct{}
}

func NewIngressCache() Cache {
	return &ingressCache{
		hostToIngressEntry: make(map[string][]ingressEntry),
		ingressMap:         make(map[string]*v1beta1.Ingress),

		reloadChannel: make(chan struct{}),
	}
}

func (ic *ingressCache) MatchRequest(r *http.Request) (namespace, name, addr string, found bool) {
	if ic.hostToIngressEntry == nil {
		klog.Errorf("hostToIngressEntry is nil")
		return "", "", "", false
	}

	hostName := getHostName(r)

	klog.Infof("request hostname: %v, path: %v", hostName, r.URL.Path)
	return ic.match(hostName, r.URL.Path)
}

func (ic *ingressCache) match(host, path string) (namespace, name, addr string, isMatch bool) {
	for h, entries := range ic.hostToIngressEntry {
		isHostMatch, _ := regexp.MatchString(fmt.Sprintf("^%s$", h), host)
		if isHostMatch {
			for _, entry := range entries {
				if strings.HasPrefix(path, entry.path.Path) {
					// special treatment for canary ingress
					if val, ok := entry.ingress.Annotations[CANARY_WEIGHT]; ok {
						canForward, err := ic.canForward(val)
						if err != nil {
							continue
						}

						if !canForward {
							klog.Infof("skip canary ingress %v this time", entry.ingress.Name)
							continue
						}
					}

					svcName := fmt.Sprintf("%s.%s.svc:%s", entry.path.Backend.ServiceName, entry.ingress.Namespace, entry.path.Backend.ServicePort.String())
					return entry.ingress.Namespace, entry.ingress.Name, svcName, true
				}
			}
		}
	}

	klog.Warningf("this request doesn't match any addr")
	return "", "", "", false
}

func (ic *ingressCache) canForward(weight string) (bool, error) {
	w, err := strconv.Atoi(weight)
	if err != nil {
		klog.Errorf("canary weight %v is not int type", weight)
		return false, err
	}

	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(100)
	klog.Infof("canary weight: %v, random: %v", weight, r)
	if r > w {
		return false, nil
	}

	return true, nil
}

func (ic *ingressCache) StartAutoUpdate(ctx context.Context, kubeClient *kubernetes.Clientset) {
	sharedInformers := informers.NewSharedInformerFactory(kubeClient, 0)
	ingressInformer := sharedInformers.Networking().V1beta1().Ingresses().Informer()
	ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ic.addIngress,
		UpdateFunc: ic.updateIngress,
		DeleteFunc: ic.deleteIngress,
	})

	stopCh := ctx.Done()
	klog.Infof("ingress informer starts to run...")
	go ingressInformer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, ingressInformer.HasSynced) {
		klog.Errorf("ingress informer sync cache failed")
		return
	}

	go wait.Until(ic.reload, 10*time.Second, stopCh)
}

func (ic *ingressCache) addIngressToMap(ingress *v1beta1.Ingress) {
	ingressKey := ingress.Namespace + "/" + ingress.Name
	klog.Infof("add ingress %v to map", ingressKey)
	ic.mutex.Lock()
	ic.ingressMap[ingressKey] = ingress
	ic.mutex.Unlock()

	ic.reloadChannel <- struct{}{}
}

func (ir *ingressCache) removeIngressFromMap(ingress *v1beta1.Ingress) {
	ingressKey := ingress.Namespace + "/" + ingress.Name
	klog.Infof("remove ingress %v from map", ingressKey)
	ir.mutex.Lock()
	delete(ir.ingressMap, ingressKey)
	ir.mutex.Unlock()
	ir.reloadChannel <- struct{}{}
}

func (ic *ingressCache) addIngress(obj interface{}) {
	ingress, ok := obj.(*v1beta1.Ingress)
	if !ok {
		klog.Errorf("expect an ingress, got an object of type %T", obj)
		return
	}

	ic.addIngressToMap(ingress)
}

func (ic *ingressCache) updateIngress(oldObj, newObj interface{}) {
	oldIngress, ok := oldObj.(*v1beta1.Ingress)
	if !ok {
		klog.Errorf("oldObj is not an ingress")
		return
	}

	newIngress, ok := newObj.(*v1beta1.Ingress)
	if !ok {
		klog.Errorf("newObj is not an ingress")
		return
	}

	// No need to update if ResourceVersion is not changed
	if oldIngress.ResourceVersion == newIngress.ResourceVersion {
		klog.V(6).Infof("no need to update because ingress is not modified.")
		return
	}

	ic.addIngressToMap(newIngress)
}

func (ic *ingressCache) deleteIngress(obj interface{}) {
	ingress, ok := obj.(*v1beta1.Ingress)
	if !ok {
		klog.Errorf("expect an ingress, got an object of type %T", obj)
		return
	}

	ic.removeIngressFromMap(ingress)
}

func (ic *ingressCache) waitForUpdates() {
	timer := time.NewTimer(50 * time.Millisecond)
	for {
		select {
		case <-timer.C:
			return
		case _, ok := <-ic.reloadChannel:
			if !ok {
				return
			}
		}
	}
}

func (ic *ingressCache) reloadHostToIngressEntry() {
	klog.Infof("start to reload the hostToIngressEntry...")
	klog.Infof("current ingress map len: %v", len(ic.ingressMap))
	ic.hostToIngressEntry = make(map[string][]ingressEntry)
	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	for _, ingress := range ic.ingressMap {
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}

			host := rule.Host
			if host == "" {
				host = "*"
			}

			for _, path := range rule.HTTP.Paths {
				ic.hostToIngressEntry[host] = append(ic.hostToIngressEntry[host], ingressEntry{ingress, path})
			}
		}
	}

	klog.Infof("reloaded hostToIngressEntry: %+v", ic.hostToIngressEntry)
	ic.sortHostToIngressEntry()
}

func (ic *ingressCache) sortHostToIngressEntry() {
	for _, paths := range ic.hostToIngressEntry {
		sort.Slice(paths, func(i, j int) bool {
			paths[i].ingress.GetAnnotations()
			_, iok := paths[i].ingress.Annotations[CANARY_WEIGHT]
			_, jok := paths[j].ingress.Annotations[CANARY_WEIGHT]
			if iok && !jok {
				return true
			} else if !iok && jok {
				return false
			} else {
				return len(paths[i].path.Path) > len(paths[j].path.Path)
			}
		})
	}
}

func (ic *ingressCache) reload() {
	klog.Info("start to reload the cache...")
	select {
	case _, ok := <-ic.reloadChannel:
		if !ok {
			return
		}

		ic.reloadHostToIngressEntry()
	default:
		return
	}
}

func getHostName(r *http.Request) string {
	hostname := r.Header.Get("Host")
	if hostname == "" {
		hostname = r.Host
	}

	if idx := strings.IndexByte(hostname, ':'); idx >= 0 {
		hostname = hostname[:idx]
	}

	return hostname
}
