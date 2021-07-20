package registry

import "github.com/alibaba/kubedl/console/backend/pkg/storage/objects/proxy"

func init() {
	newObjectBackends = append(newObjectBackends, proxy.NewProxyObjectBackend)
}
