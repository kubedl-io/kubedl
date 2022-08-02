package plugins

import (
	"time"

	"github.com/alibaba/kubedl/pkg/jobcoordinator"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/plugins/priority"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/plugins/quota"
)

const (
	DefaultSchedulePeriod = 100 * time.Millisecond
)

func NewCoordinatorConfiguration() jobcoordinator.CoordinatorConfiguration {
	cfg := jobcoordinator.CoordinatorConfiguration{
		SchedulePeriod:    DefaultSchedulePeriod,
		TenantPlugin:      quota.Name,
		ScorePlugins:      []string{priority.Name},
		FilterPlugins:     []string{quota.Name},
		PreDequeuePlugins: []string{quota.Name},
	}

	return cfg
}

func NewPluginsRegistry() map[string]jobcoordinator.PluginFactory {
	return map[string]jobcoordinator.PluginFactory{
		priority.Name: priority.New,
		quota.Name:    quota.New,
	}
}

type Registry map[string]jobcoordinator.PluginFactory
