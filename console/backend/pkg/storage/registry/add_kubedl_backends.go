package registry

import "github.com/alibaba/kubedl/pkg/storage/backends/registry"

func init() {
	newObjectBackends = append(newObjectBackends, registry.NewObjectBackends...)
	newEventBackends = append(newEventBackends, registry.NewEventBackends...)
}
