package registry

import "github.com/alibaba/kubedl/console/backend/pkg/storage/objects/apiserver"

func init() {
	newObjectBackends = append(newObjectBackends, apiserver.NewAPIServerObjectBackend)
}
