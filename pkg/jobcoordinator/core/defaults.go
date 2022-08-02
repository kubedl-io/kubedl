package core

import (
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/pkg/jobcoordinator"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/plugins/quota"
)

func setDefaultCoordinatorPlugins(co *coordinator, client client.Client, recorder record.EventRecorder) {
	// tenant plugin is required.
	co.tenantPlugin = quota.New(client, recorder).(jobcoordinator.TenantPlugin)
}
