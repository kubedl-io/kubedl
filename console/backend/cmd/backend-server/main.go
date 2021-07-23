package main

import (
	"flag"
	"github.com/alibaba/kubedl/console/backend/pkg/storage"

	"github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/routers"
	"github.com/alibaba/kubedl/pkg/storage/backends/registry"
)

func main() {
	flag.Parse()
	client.Init()
	registry.RegisterStorageBackends()
	storage.RegisterObjectBackend()
	r := routers.InitRouter()

	client.Start()
	_ = r.Run(":9090")
}
