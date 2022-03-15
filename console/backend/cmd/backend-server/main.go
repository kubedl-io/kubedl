package main

import (
	"context"
	"flag"

	"github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/routers"
	"github.com/alibaba/kubedl/console/backend/pkg/storage"
)

func main() {
	flag.Parse()
	client.Init()
	storage.RegisterStorageBackends()
	r := routers.InitRouter()

	client.Start(context.Background())
	_ = r.Run(":9090")
}
