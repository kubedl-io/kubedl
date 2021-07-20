package registry

import "github.com/alibaba/kubedl/console/backend/pkg/storage/events/apiserver"

func init() {
	newEventBackends = append(newEventBackends, apiserver.NewAPIServerEventBackend)
}
