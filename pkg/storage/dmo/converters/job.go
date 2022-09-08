/*
Copyright 2020 The Alibaba Authors.

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

package converters

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"

	inferencev1alpha1 "github.com/alibaba/kubedl/apis/inference/v1alpha1"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/tenancy"
)

const (
	RemarkEnableTensorBoard = "EnableTensorBoard"
)

// ConvertJobToDMOJob converts a native job object to dmo job.
func ConvertJobToDMOJob(job metav1.Object, kind string, specs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, region string) (*dmo.Job, error) {
	klog.V(5).Infof("[ConvertJobToDMOJob] kind: %s, job: %s/%s", kind, job.GetNamespace(), job.GetName())
	dmoJob := dmo.Job{
		Name:       job.GetName(),
		Namespace:  job.GetNamespace(),
		JobID:      string(job.GetUID()),
		Version:    job.GetResourceVersion(),
		Kind:       kind,
		Resources:  "",
		GmtCreated: job.GetCreationTimestamp().Time,
	}

	if region != "" {
		dmoJob.DeployRegion = &region
	}

	if tn, err := tenancy.GetTenancy(job); err == nil && tn != nil {
		dmoJob.Tenant = &tn.Tenant
		dmoJob.Owner = &tn.User
		if dmoJob.DeployRegion == nil && tn.Region != "" {
			dmoJob.DeployRegion = &tn.Region
		}
	} else {
		dmoJob.Tenant = pointer.StringPtr("")
		dmoJob.Owner = pointer.StringPtr("")
	}

	dmoJob.Status = v1.JobCreated
	if condLen := len(jobStatus.Conditions); condLen > 0 {
		dmoJob.Status = jobStatus.Conditions[condLen-1].Type
	}

	if runningCond := util.GetCondition(*jobStatus, v1.JobRunning); runningCond != nil {
		dmoJob.GmtJobRunning = &runningCond.LastTransitionTime.Time
	}

	if finishTime := jobStatus.CompletionTime; finishTime != nil {
		dmoJob.GmtJobFinished = &finishTime.Time
	}

	dmoJob.Deleted = util.IntPtr(0)
	dmoJob.IsInEtcd = util.IntPtr(1)

	resources := computeJobResources(specs)
	resourcesBytes, err := json.Marshal(&resources)
	if err != nil {
		return nil, err
	}
	dmoJob.Resources = string(resourcesBytes)

	if _, ok := job.GetAnnotations()[v1.AnnotationTensorBoardConfig]; ok {
		enableTB := RemarkEnableTensorBoard
		dmoJob.Remark = &enableTB
	}
	return &dmoJob, nil
}

// ExtractTypedJobInfos extract common-api struct and infos from different typed job objects.
func ExtractTypedJobInfos(job metav1.Object) (kind string, spec map[v1.ReplicaType]*v1.ReplicaSpec, status v1.JobStatus, err error) {
	switch typed := job.(type) {
	case *trainingv1alpha1.TFJob:
		return trainingv1alpha1.TFJobKind, typed.Spec.TFReplicaSpecs, typed.Status, nil
	case *trainingv1alpha1.PyTorchJob:
		return trainingv1alpha1.PyTorchJobKind, typed.Spec.PyTorchReplicaSpecs, typed.Status, nil
	case *trainingv1alpha1.XGBoostJob:
		return trainingv1alpha1.XGBoostJobKind, typed.Spec.XGBReplicaSpecs, typed.Status.JobStatus, nil
	case *trainingv1alpha1.XDLJob:
		return trainingv1alpha1.XDLJobKind, typed.Spec.XDLReplicaSpecs, typed.Status, nil
	case *inferencev1alpha1.ElasticBatchJob:
		return inferencev1alpha1.ElasticBatchJobKind, typed.Spec.ElasticBatchReplicaSpecs, typed.Status, nil
	}
	return "", nil, v1.JobStatus{}, fmt.Errorf("unkonwn job kind, %s/%s", job.GetNamespace(), job.GetName())
}

type replicaResources struct {
	Resources corev1.ResourceRequirements `json:"resources"`
	Replicas  int32                       `json:"replicas"`
}

type replicaResourcesMap map[v1.ReplicaType]replicaResources

func computeJobResources(specs map[v1.ReplicaType]*v1.ReplicaSpec) replicaResourcesMap {
	resources := make(replicaResourcesMap)
	for rtype, spec := range specs {
		specResources := computePodResources(&spec.Template.Spec)
		rr := replicaResources{Resources: specResources}
		if spec.Replicas != nil {
			rr.Replicas = *spec.Replicas
		}
		resources[rtype] = rr
	}
	return resources
}
