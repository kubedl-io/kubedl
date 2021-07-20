package main

import (
	"flag"
	"github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/routers"
	"github.com/alibaba/kubedl/console/backend/pkg/storage/registry"
)

func main() {
	flag.Parse()
	client.Init()
	registry.RegisterStorageBackends()
	r := routers.InitRouter()

	client.Start()
	_ = r.Run(":9090")
}
