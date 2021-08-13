package registry

import (
	"github.com/alibaba/kubedl/pkg/gang_schedule/volcano_scheduler"
)

func init() {
	NewGangSchedulers = append(NewGangSchedulers, volcano_scheduler.NewVolcanoScheduler)
}
