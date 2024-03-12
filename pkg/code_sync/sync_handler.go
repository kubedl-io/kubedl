package code_sync

import (
	"encoding/json"
	"path"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	v1 "k8s.io/api/core/v1"
)

const (
	DefaultCodeRootPath = "/code"
)

type CodeSyncHandler interface {
	InitContainer(optsConfig []byte, mountVolume *v1.Volume) (c *v1.Container, codePath string, err error)
}

func InjectCodeSyncInitContainers(specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, gitSyncConfig *commonv1.GitSyncOptions) error {
	if cfg := gitSyncConfig; cfg != nil {
		optsConfig, err := json.Marshal(gitSyncConfig)
		if err != nil {
			return err
		}
		if err = injectCodeSyncInitContainer(optsConfig, &gitSyncHandler{}, specs, &v1.Volume{
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
