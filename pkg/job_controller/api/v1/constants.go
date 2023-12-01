package v1

import v1 "k8s.io/api/core/v1"

const (
	// ReplicaIndexLabel represents the label key for the replica-index, e.g. the value is 0, 1, 2.. etc
	ReplicaIndexLabel = "replica-index"

	// ReplicaTypeLabel represents the label key for the replica-type, e.g. the value is ps , worker etc.
	ReplicaTypeLabel = "replica-type"

	// ReplicaNameLabel represents the label key for the replica-name, the value is replica name.
	ReplicaNameLabel = "replica-name"

	// GroupNameLabel represents the label key for group name, e.g. the value is kubeflow.org
	GroupNameLabel = "group-name"

	// JobNameLabel represents the label key for the job name, the value is job name
	JobNameLabel = "job-name"

	// JobRoleLabel represents the label key for the job role, e.g. the value is master
	JobRoleLabel = "job-role"
)

// Constant label/annotation keys for job configuration.
const (
	KubeDLPrefix = "kubedl.io"

	// AnnotationGitSyncConfig annotate git sync configurations.
	AnnotationGitSyncConfig = KubeDLPrefix + "/git-sync-config"
	// AnnotationTenancyInfo annotate tenancy information.
	AnnotationTenancyInfo = KubeDLPrefix + "/tenancy"
	// AnnotationNetworkMode annotate job network mode.
	AnnotationNetworkMode = KubeDLPrefix + "/network-mode"
	// AnnotationEnableElasticTraining indicates job enables elastic training.
	AnnotationEnableElasticTraining = KubeDLPrefix + "/enable-elastic-training"
	// AnnotationElasticScaleState indicates current progress of elastic scaling (inflight | done)
	AnnotationElasticScaleState = KubeDLPrefix + "/scale-state"

	// AnnotationTensorBoardConfig annotate tensorboard configurations.
	AnnotationTensorBoardConfig = KubeDLPrefix + "/tensorboard-config"
	// ReplicaTypeTensorBoard is the type for TensorBoard.
	ReplicaTypeTensorBoard ReplicaType = "TensorBoard"
	// ResourceNvidiaGPU is the key of gpu type in labels
	ResourceNvidiaGPU v1.ResourceName = "nvidia.com/gpu"
)

const (
	// LabelInferenceName represents the inference service name.
	LabelInferenceName = KubeDLPrefix + "/inference-name"
	// LabelPredictorName represents the predictor name of served model.
	LabelPredictorName = KubeDLPrefix + "/predictor-name"
	// LabelModelVersion represents the model version value for inference role.
	LabelModelVersion = KubeDLPrefix + "/model-version"
	// LabelCronName indicates the name of cron who created this job.
	LabelCronName = KubeDLPrefix + "/cron-name"
	// LabelGangSchedulingJobName indicates name of gang scheduled job.
	LabelGangSchedulingJobName = KubeDLPrefix + "/gang-job-name"
	// LabelGeneration indicates the generation of this job referenced to.
	LabelGeneration = KubeDLPrefix + "/job-generation"
)

type ContextKey string

const (
	// ContextFailedPodContents collects failed pod exit codes while with its failed
	// reason if they are not retryable.
	ContextFailedPodContents ContextKey = KubeDLPrefix + "/failed-pod-contents"

	// ContextHostNetworkPorts is the key for passing selected host-ports, value is
	// a map object [replica-index: port].
	ContextHostNetworkPorts ContextKey = KubeDLPrefix + "/hostnetwork-ports"
)

const (
	FinalizerPreemptProtector = KubeDLPrefix + "/preempt-protector"
)

const (
	// JobReplicaTypeAIMaster means the AIMaster role for all job
	JobReplicaTypeAIMaster ReplicaType = "AIMaster"
)

// NetworkMode defines network mode for intra job communicating.
type NetworkMode string

const (
	// HostNetworkMode indicates that replicas use host-network to communicate with each other.
	HostNetworkMode NetworkMode = "host"
)

const (
	ElasticScaleInflight = "inflight"
	ElasticScaleDone     = "done"
)
