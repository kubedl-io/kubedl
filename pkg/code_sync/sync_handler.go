package code_sync

import (
	"path"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultCodeRootPath = "/code"
)

type CodeSyncHandler interface {
	InitContainer(optsConfig []byte, mountVolume *v1.Volume) (c *v1.Container, codePath string, err error)
}

type SyncOptions struct {
	// Code source address.(required)
	Source string `json:"source"`
	// Image contains toolkits to execute syncing code.
	Image string `json:"image,omitempty"`
	// Code root/destination directory path.
	// Root: the path to save downloaded files.
	// Dest: the name of (a symlink to) a directory in which to check-out files
	RootPath string `json:"rootPath,omitempty"`
	DestPath string `json:"destPath,omitempty"`
	// User-customized environment variables.
	Envs []v1.EnvVar `json:"envs,omitempty"`
}

func InjectCodeSyncInitContainers(metaObj metav1.Object, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {
	var err error

	if cfg, ok := metaObj.GetAnnotations()[apiv1.AnnotationGitSyncConfig]; ok {
		if err = injectCodeSyncInitContainer([]byte(cfg), &gitSyncHandler{}, specs, &v1.Volume{
			Name:         "git-sync",
			VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
		}); err != nil {
			return err
		}
	}

	// TODO(SimonCqk): support other sources.

	return nil
}

func injectCodeSyncInitContainer(optsConfig []byte, handler CodeSyncHandler, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, mountVolume *v1.Volume) error {
	initContainer, dest, err := handler.InitContainer(optsConfig, mountVolume)
	if err != nil {
		return err
	}

	for _, spec := range specs {
		initContainer.Resources = *spec.Template.Spec.Containers[0].Resources.DeepCopy()
		spec.Template.Spec.InitContainers = append(spec.Template.Spec.InitContainers, *initContainer)

		// Inject volumes and volume mounts into main containers.
		spec.Template.Spec.Volumes = append(spec.Template.Spec.Volumes, *mountVolume)
		for idx := range spec.Template.Spec.Containers {
			container := &spec.Template.Spec.Containers[idx]
			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				Name:      mountVolume.Name,
				ReadOnly:  false,
				MountPath: path.Join(container.WorkingDir, dest),
				SubPath:   dest,
			})
		}
	}
	return nil
}
