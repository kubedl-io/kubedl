/*
Copyright 2019 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package registry

import (
	"sync"

	"github.com/alibaba/kubedl/pkg/gang_schedule"
	"k8s.io/klog"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var (
	NewGangSchedulers []gang_schedule.NewGangScheduler
	defaultRegistry   = Registry{registry: make(map[string]gang_schedule.GangScheduler)}
)

func RegisterGangSchedulers(mgr controllerruntime.Manager) {
	for _, newer := range NewGangSchedulers {
		scheduler := newer(mgr)
		klog.Infof("register gang scheduler %s", scheduler.Name())
		defaultRegistry.Add(scheduler)
	}
}

func Add(scheduler gang_schedule.GangScheduler) {
	defaultRegistry.Add(scheduler)
}

func Get(name string) gang_schedule.GangScheduler {
	return defaultRegistry.Get(name)
}

func Remove(name string) {
	defaultRegistry.Remove(name)
}

type Registry struct {
	lock     sync.Mutex
	registry map[string]gang_schedule.GangScheduler
}

func (r *Registry) Add(scheduler gang_schedule.GangScheduler) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.registry[scheduler.Name()] = scheduler
}

func (r *Registry) Get(name string) gang_schedule.GangScheduler {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.registry[name]
}

func (r *Registry) Remove(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.registry, name)
}
