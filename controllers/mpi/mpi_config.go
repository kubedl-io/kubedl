/*
Copyright 2021 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mpi

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
)

const (
	kubectlMountPath    = "/opt/kube"
	kubectlTargetDirEnv = "TARGET_DIR"
	kubectlVolumeName   = "kubectl-volume"
	kubexecScriptName   = "kubexec.sh"
	hostfileName        = "hostfile"

	configVolumeName      = "mpi-config-volume"
	configVolumeMountPath = "/etc/mpi"
)

var (
	launcherSuffix = "-" + strings.ToLower(string(training.MPIReplicaTypeLauncher))
	workerSuffix   = "-" + strings.ToLower(string(training.MPIReplicaTypeWorker))
)

func (r *MPIJobReconciler) getOrCreateJobConfig(mpiJob *training.MPIJob, workerReplicas int32, launcherRunsWorkload bool) (*corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{}
	err := r.Client.Get(context.Background(), types.NamespacedName{
		Namespace: mpiJob.Namespace,
		Name:      jobConfigName(mpiJob),
	}, &cm)
	if err != nil && errors.IsNotFound(err) {
		// If the ConfigMap doesn't exist, we'll create it.
		err = r.Client.Create(context.Background(), newJobConfigMap(mpiJob, workerReplicas, launcherRunsWorkload))
	}

	if err != nil {
		return nil, err
	}
	return &cm, nil
}

// newConfigMap creates a new ConfigMap containing configurations for an MPIJob
// resource. It also sets the appropriate OwnerReferences on the resource so
// handleObject can discover the MPIJob resource that 'owns' it.
func newJobConfigMap(mpiJob *training.MPIJob, workerReplicas int32, launcherRunsWorkload bool) *corev1.ConfigMap {
	kubexec := fmt.Sprintf(`#!/bin/sh
set -x
POD_NAME=$1
shift
%s/kubectl exec ${POD_NAME}`, kubectlMountPath)
	if len(mpiJob.Spec.MainContainer) > 0 {
		kubexec = fmt.Sprintf("%s --container %s", kubexec, mpiJob.Spec.MainContainer)
	}
	kubexec = fmt.Sprintf("%s -- /bin/sh -c \"$*\"", kubexec)

	// If no processing unit is specified, default to 1 slot.
	slots := 1
	if mpiJob.Spec.SlotsPerWorker != nil {
		slots = int(*mpiJob.Spec.SlotsPerWorker)
	}
	var buffer bytes.Buffer
	if launcherRunsWorkload {
		buffer.WriteString(fmt.Sprintf("%s%s-0 slots=%d\n", mpiJob.Name, launcherSuffix, slots))
	}
	for i := 0; i < int(workerReplicas); i++ {
		// If MPIDistributionType specified, the format of the hostfile file is inconsistent as
		// for different MPI framework.
		// For Intel MPI and MVAPICH2, use ":" syntax to indicate how many operating slots the current node has.
		// But for Open MPI, use "slots=" syntax to achieve this function.
		if mpiJob.Spec.MPIJobLegacySpec != nil && mpiJob.Spec.MPIJobLegacySpec.LegacyV1Alpha2 != nil &&
			mpiJob.Spec.MPIJobLegacySpec.LegacyV1Alpha2.MPIDistribution != nil &&
			(*mpiJob.Spec.MPIJobLegacySpec.LegacyV1Alpha2.MPIDistribution == training.MPIDistributionTypeIntelMPI ||
				*mpiJob.Spec.MPIJobLegacySpec.LegacyV1Alpha2.MPIDistribution == training.MPIDistributionTypeMPICH) {
			buffer.WriteString(fmt.Sprintf("%s%s-%d:%d\n", mpiJob.Name, workerSuffix, i, slots))
		} else {
			buffer.WriteString(fmt.Sprintf("%s%s-%d slots=%d\n", mpiJob.Name, workerSuffix, i, slots))
		}
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobConfigName(mpiJob),
			Namespace: mpiJob.Namespace,
			Labels: map[string]string{
				"app": mpiJob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(mpiJob, training.SchemeGroupVersion.WithKind(training.MPIJobKind)),
			},
		},
		Data: map[string]string{
			hostfileName:      buffer.String(),
			kubexecScriptName: kubexec,
		},
	}
}

func jobConfigName(mpiJob *training.MPIJob) string {
	return fmt.Sprintf("%s-config", mpiJob.Name)
}
