package torchelastic

import (
	"github.com/alibaba/kubedl/controllers/torchelastic/job"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	controllerName = "ElasticScalingController"
)

func SetupWithManager(mgr ctrl.Manager) error {
	// New torch elastic controller.
	// period represents the time elastic scaling loop repeats.
	// count represents the length of training metrics collection for each scale replicas.
	torchElasticController := job.NewTorchElasticController(mgr, 30, 5)

	if err := torchElasticController.SetupWithManager(mgr); err != nil {
		return err
	}
	return nil

}
