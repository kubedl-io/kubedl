package utils

import apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

const (
	// Expanded object statuses only used in persistent layer.
	PodStopped = "Stopped"
	// JobStopped means the job has been stopped manually by user,
	// a job can be stopped only when job has not reached a final
	// state(Succeed/Failed).
	JobStopped apiv1.JobConditionType = "Stopped"
	// JobStopping is a intermediate state before truly Stopped.
	JobStopping apiv1.JobConditionType = "Stopping"
)
