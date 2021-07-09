package constants

import "flag"

const (
	KubeDLSystemNamespace = "kubedl-system"
	ApiV1Routes = "/api/v1"
)

var (
	ConfigMapName = "kubedl-config"
)

func init() {
	flag.StringVar(&ConfigMapName, "config-name", "kubedl-config", "kubedl common configmap name in kubedl-system namespace.")
}