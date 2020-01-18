package code_sync

import (
	"encoding/json"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const (
	defaultGitSyncImage = "kubedl/git-sync:v1"
)

var _ CodeSyncHandler = &gitSyncHandler{}

type gitSyncOptions struct {
	SyncOptions `json:",inline"`

	// All fields down below are optional.

	// Git repository settings for user to specify.
	Branch   string `json:"branch,omitempty"`
	Revision string `json:"revision,omitempty"`
	Depth    string `json:"depth,omitempty"`
	// Max consecutive failures allowed.
	MaxFailures int `json:"maxFailures,omitempty"`
	// SSH settings for users to use git in ssh pattern.
	SSH     bool   `json:"ssh,omitempty"`
	SSHFile string `json:"sshFile,omitempty"`
	// User-customized account settings.
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

type gitSyncHandler struct{}

func (h *gitSyncHandler) InitContainer(optsConfig []byte, mountVolume *v1.Volume) (*v1.Container, string, error) {
	opts := gitSyncOptions{}
	if err := json.Unmarshal(optsConfig, &opts); err != nil {
		return nil, "", err
	}
	setDefaultSyncOpts(&opts)
	setSyncOptsEnvs(&opts)

	container := v1.Container{
		Name:            "git-sync-code",
		Image:           opts.Image,
		Env:             opts.Envs,
		ImagePullPolicy: v1.PullIfNotPresent,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      mountVolume.Name,
				ReadOnly:  false,
				MountPath: opts.RootPath,
			},
		},
	}
	return &container, opts.DestPath, nil
}

func setDefaultSyncOpts(opts *gitSyncOptions) {
	if opts.RootPath == "" {
		opts.RootPath = DefaultCodeRootPath
	}
	// Default as project name parsed from git path.
	if opts.DestPath == "" {
		parts := strings.Split(strings.Trim(opts.Source, "/"), "/")
		opts.DestPath = parts[len(parts)-1]
		if strings.HasSuffix(opts.DestPath, ".git") {
			opts.DestPath = opts.DestPath[:len(opts.DestPath)-4]
		}
	}
	if opts.Image == "" {
		opts.Image = defaultGitSyncImage
	}
	if opts.MaxFailures == 0 {
		opts.MaxFailures = 3
	}
}

func setSyncOptsEnvs(opts *gitSyncOptions) {
	opts.Envs = append(opts.Envs, v1.EnvVar{
		Name:  "GIT_SYNC_REPO",
		Value: opts.Source,
	})
	// Critical: if it's false the init container will never exit.
	opts.Envs = append(opts.Envs, v1.EnvVar{
		Name:  "GIT_SYNC_ONE_TIME",
		Value: "true",
	})
	if opts.MaxFailures >= 0 {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_MAX_SYNC_FAILURES",
			Value: strconv.Itoa(opts.MaxFailures),
		})
	}
	if opts.Branch != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_BRANCH",
			Value: opts.Branch,
		})
	}
	if opts.Revision != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_REV",
			Value: opts.Revision,
		})
	}
	if opts.Depth != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_DEPTH",
			Value: opts.Depth,
		})
	}
	if opts.RootPath != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_ROOT",
			Value: opts.RootPath,
		})
	}
	if opts.DestPath != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_DEST",
			Value: opts.DestPath,
		})
	}
	if opts.SSH {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_SSH",
			Value: "true",
		})
	}
	if opts.SSH && opts.SSHFile != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SSH_KEY_FILE",
			Value: opts.SSHFile,
		})
	}
	if opts.User != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_USERNAME",
			Value: opts.User,
		})
	}
	if opts.Password != "" {
		opts.Envs = append(opts.Envs, v1.EnvVar{
			Name:  "GIT_SYNC_PASSWORD",
			Value: opts.Password,
		})
	}
}
