package core

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	coordinatorQueuePendingJobsCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubedl_tenant_queue_jobs_pending_count",
		Help: "Counter for accounting pending jobs in each tenant queue",
	}, []string{"queue"})
)
