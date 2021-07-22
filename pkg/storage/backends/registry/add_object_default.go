package registry

import defaultBackend "github.com/alibaba/kubedl/pkg/storage/backends/objects/default"

func init() {
	NewObjectBackends = append(NewObjectBackends, defaultBackend.NewDefaultBackendService)
}
