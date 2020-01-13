package code_sync

import (
	"encoding/json"
	"fmt"
	"path"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	v1 "k8s.io/api/core/v1"
)

const (
	DefaultCodeRootPath = "/code"
)

type CodeSyncMode string

const (
	GitSyncMode  CodeSyncMode = "git"
	HttpSyncMode CodeSyncMode = "http"
	HDFDSyncMode CodeSyncMode = "hdfs"
)

type CodeSyncHandler interface {
	GenerateInitContainer(optsConfig []byte, mountVolumeName string) (v1.Container, SyncOptions, error)
}

type mode struct {
	// Code sync mode: git, http, hdfs...(required)
	Mode CodeSyncMode `json:"mode"`
}

type SyncOptions struct {
	mode `json:",inline"`
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

func InjectCodeSyncInitContainers(config string, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {
	var (
		handler CodeSyncHandler
		mode    mode
	)

	cfgBytes := []byte(config)
	err := json.Unmarshal(cfgBytes, &mode)
	if err != nil {
		return err
	}

	if handler = newSyncHandler(&mode); handler == nil {
		return fmt.Errorf("unspported sync mode: %s", mode.Mode)
	}

	syncVolume := v1.Volume{
		Name:         "code-sync",
		VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
	}

	initContainer, opts, err := handler.GenerateInitContainer(cfgBytes, syncVolume.Name)
	if err != nil {
		return err
	}

	for _, spec := range specs {
		initContainer.Resources = *spec.Template.Spec.Containers[0].Resources.DeepCopy()
		spec.Template.Spec.InitContainers = append(spec.Template.Spec.InitContainers, initContainer)

		// Inject volumes and volume mounts into main containers.
		spec.Template.Spec.Volumes = append(spec.Template.Spec.Volumes, syncVolume)
		for idx := range spec.Template.Spec.Containers {
			container := &spec.Template.Spec.Containers[idx]
			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				Name:      syncVolume.Name,
				ReadOnly:  false,
				MountPath: path.Join(container.WorkingDir, opts.DestPath),
			})
		}
	}
	return nil
}

func newSyncHandler(m *mode) CodeSyncHandler {
	switch m.Mode {
	case GitSyncMode:
		return &gitSyncHandler{}
	}
	return nil
}
