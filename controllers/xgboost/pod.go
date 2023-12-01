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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"
)

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *XgboostJobReconciler) GetPodsForJob(job client.Object) ([]*corev1.Pod, error) {
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(job.GetName()),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podList := &corev1.PodList{}
	err := r.List(context.Background(), podList, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	return r.ctrl.AdoptAndClaimPods(job, podList)
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

	masterAddr := commonutil.GenGeneralName(xgboostjob.Name, strings.ToLower(string(v1alpha1.XGBoostReplicaTypeMaster)), strconv.Itoa(0))
	masterPort, err := job_controller.GetPortFromJob(xgboostjob.Spec.XGBReplicaSpecs, v1alpha1.XGBoostReplicaTypeMaster, v1alpha1.XGBoostJobDefaultContainerName, v1alpha1.XGBoostJobDefaultContainerPortName)
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
