package registry

import "github.com/alibaba/kubedl/pkg/storage/backends/objects/apiserver"

func init() {
	NewObjectBackends = append(NewObjectBackends, apiserver.NewAPIServerBackendService)
}
