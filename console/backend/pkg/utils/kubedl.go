package utils

import (
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	corev1 "k8s.io/api/core/v1"
)

func IsKubedlManagedNamespace(namespace *corev1.Namespace) bool {
	if _, ok := namespace.Labels[model.WorkspaceKubeDLLabel]; ok {
		return true
	}
	return false
}
