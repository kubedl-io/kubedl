/*

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

package xgboostjob

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"
)

// CreatePod creates the pod
func (r *XgboostJobReconciler) CreatePod(job interface{}, pod *corev1.Pod) error {
	xgboostjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	logrus.Info("Creating pod ", " Controller name : ", xgboostjob.GetName(), " Pod name: ", pod.Namespace+"/"+pod.Name)

	if err := r.Create(context.Background(), pod); err != nil {
		log.Info("Error building a pod via XGBoost operator: %s", err.Error())
		return err
	}
	return nil
}

// DeletePod deletes the pod
func (r *XgboostJobReconciler) DeletePod(job interface{}, pod *corev1.Pod) error {
	xgboostjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	if err := r.Delete(context.Background(), pod); err != nil {
		r.recorder.Eventf(xgboostjob, corev1.EventTypeWarning, job_controller.FailedDeletePodReason, "Error deleting: %v", err)
		return err
	}
	r.recorder.Eventf(xgboostjob, corev1.EventTypeNormal, job_controller.SuccessfulDeletePodReason, "Deleted pod: %v", pod.Name)

	logrus.Info("Controller: ", xgboostjob.GetName(), " delete pod: ", pod.Namespace+"/"+pod.Name)

	return nil
}

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *XgboostJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(job.GetName()),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	pods := util.ToPodPointerList(podlist.Items)
	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := job_controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := r.GetJobFromAPIClient(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := controller.NewPodControllerRefManager(job_controller.NewPodControl(r.Client, r.recorder), job, selector, r.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}

// Set the pod env set for XGBoost Rabit Tracker and worker
func SetPodEnv(job interface{}, podTemplate *corev1.PodTemplateSpec, index string) error {
	xgboostjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	masterAddr := job_controller.GenGeneralName(xgboostjob.Name, strings.ToLower(string(v1alpha1.XGBoostReplicaTypeMaster)), strconv.Itoa(0))
	masterPort, err := job_controller.GetPortFromJob(xgboostjob.Spec.XGBReplicaSpecs, v1alpha1.XGBoostReplicaTypeMaster, v1alpha1.DefaultContainerName, v1alpha1.DefaultContainerPortName)
	if err != nil {
		return err
	}

	totalReplicas := int(k8sutil.GetTotalReplicas(xgboostjob.Spec.XGBReplicaSpecs))

	for i := range podTemplate.Spec.Containers {
		if len(podTemplate.Spec.Containers[i].Env) == 0 {
			podTemplate.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
		}
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_PORT",
			Value: strconv.Itoa(int(masterPort)),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "MASTER_ADDR",
			Value: masterAddr,
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "WORLD_SIZE",
			Value: strconv.Itoa(totalReplicas),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "RANK",
			Value: strconv.Itoa(rank),
		})
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "0",
		})
	}

	return nil
}
