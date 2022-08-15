package job

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ElasticController implementations
type ElasticController interface {
	SetupWithManager(mgr ctrl.Manager) error
}

var _ ElasticController = &TorchElasticController{}

type newJobElasticController func(mgr ctrl.Manager, period, count int) ElasticController

var (
	log               = logf.Log.WithName("job-elastic-controller")
	jobElasticCtrlMap = make(map[runtime.Object]newJobElasticController)
)
