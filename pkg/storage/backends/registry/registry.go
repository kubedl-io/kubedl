/*
Copyright 2020 The Alibaba Authors.

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
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"sync"

	"k8s.io/klog"
)

var (
	NewObjectBackends      []func() backends.ObjectStorageBackend
	NewEventBackends       []func() backends.EventStorageBackend
	defaultBackendRegistry = NewBackendRegistry()
)

func RegisterStorageBackends() {
	for idx := range NewObjectBackends {
		b := NewObjectBackends[idx]()
		klog.Infof("register new object backend: %s", b.Name())
		AddObjectBackend(b)
	}
	for idx := range NewEventBackends {
		b := NewEventBackends[idx]()
		klog.Infof("register new event backend: %s", b.Name())
		AddEventBackend(b)
	}
}

func NewBackendRegistry() *Registry {
	return &Registry{
		objectBackends: make(map[string]backends.ObjectStorageBackend),
		eventBackends:  make(map[string]backends.EventStorageBackend),
	}
}

func AddObjectBackend(objBackend backends.ObjectStorageBackend) {
	defaultBackendRegistry.AddObjectBackend(objBackend)
}

func GetObjectBackend(name string) backends.ObjectStorageBackend {
	return defaultBackendRegistry.GetObjectBackend(name)
}

func RemoveObjectBackend(name string) {
	defaultBackendRegistry.RemoveObjectBackend(name)
}

func AddEventBackend(eventBackend backends.EventStorageBackend) {
	defaultBackendRegistry.AddEventBackend(eventBackend)
}

func GetEventBackend(name string) backends.EventStorageBackend {
	return defaultBackendRegistry.GetEventBackend(name)
}

func RemoveEventBackend(name string) {
	defaultBackendRegistry.RemoveEventBackend(name)
}

type Registry struct {
	lock           sync.Mutex
	objectBackends map[string]backends.ObjectStorageBackend
	eventBackends  map[string]backends.EventStorageBackend
}

func (r *Registry) AddObjectBackend(objBackend backends.ObjectStorageBackend) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.objectBackends[objBackend.Name()] = objBackend
}

func (r *Registry) GetObjectBackend(name string) backends.ObjectStorageBackend {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.objectBackends[name]
}

func (r *Registry) RemoveObjectBackend(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.objectBackends, name)
}

func (r *Registry) AddEventBackend(eventBackend backends.EventStorageBackend) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.eventBackends[eventBackend.Name()] = eventBackend
}

func (r *Registry) GetEventBackend(name string) backends.EventStorageBackend {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.eventBackends[name]
}

func (r *Registry) RemoveEventBackend(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.eventBackends, name)
}
