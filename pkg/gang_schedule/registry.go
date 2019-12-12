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

package gang_schedule

import (
	"sync"
)

var (
	defaultRegistry = Registry{registry: make(map[string]GangScheduler)}
)

func Add(scheduler GangScheduler) {
	defaultRegistry.Add(scheduler)
}

func Get(name string) GangScheduler {
	return defaultRegistry.Get(name)
}

func Remove(name string) {
	defaultRegistry.Remove(name)
}

type Registry struct {
	lock     sync.Mutex
	registry map[string]GangScheduler
}

func (r *Registry) Add(scheduler GangScheduler) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.registry[scheduler.Name()] = scheduler
}

func (r *Registry) Get(name string) GangScheduler {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.registry[name]
}

func (r *Registry) Remove(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.registry, name)
}
