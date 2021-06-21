package constants

import "flag"

const (
	DLCSystemNamespace = "kubedl-system"
	ApiV1Routes = "/api/v1"
)

var (
	ConfigMapName = "kubedl-config"
)

func init() {
	flag.StringVar(&ConfigMapName, "config-name", "kubedl-config", "dlc common configmap name in kubedl-system namespace.")
}