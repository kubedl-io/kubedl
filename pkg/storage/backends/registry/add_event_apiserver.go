package registry

import "github.com/alibaba/kubedl/pkg/storage/backends/events/apiserver"

func init() {
	NewEventBackends = append(NewEventBackends, apiserver.NewAPIServerEventBackend)
}
