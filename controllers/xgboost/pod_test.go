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
	"github.com/alibaba/kubedl/api/xgboost/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func NewXGBoostJobWithMaster(worker int) *v1alpha1.XGBoostJob {
	job := NewXGoostJob(worker)
	master := int32(1)
	masterReplicaSpec := &v1.ReplicaSpec{
		Replicas: &master,
		Template: NewXGBoostReplicaSpecTemplate(),
	}
	job.Spec.XGBReplicaSpecs[v1.ReplicaType(v1alpha1.XGBoostReplicaTypeMaster)] = masterReplicaSpec
	return job
}

func NewXGoostJob(worker int) *v1alpha1.XGBoostJob {

	job := &v1alpha1.XGBoostJob{
		TypeMeta: metav1.TypeMeta{
			Kind: v1alpha1.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-xgboostjob",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.XGBoostJobSpec{
			XGBReplicaSpecs: make(map[v1.ReplicaType]*v1.ReplicaSpec),
		},
	}

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &v1.ReplicaSpec{
			Replicas: &worker,
			Template: NewXGBoostReplicaSpecTemplate(),
		}
		job.Spec.XGBReplicaSpecs[v1.ReplicaType(v1alpha1.XGBoostReplicaTypeWorker)] = workerReplicaSpec
	}

	return job
}

func NewXGBoostReplicaSpecTemplate() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  v1alpha1.DefaultContainerName,
					Image: "test-image-for-kubeflow-xgboost-operator:latest",
					Args:  []string{"Fake", "Fake"},
					Ports: []corev1.ContainerPort{
						corev1.ContainerPort{
							Name:          v1alpha1.DefaultContainerPortName,
							ContainerPort: v1alpha1.DefaultPort,
						},
					},
				},
			},
		},
	}
}

func TestClusterSpec(t *testing.T) {
	type tc struct {
		job                 *v1alpha1.XGBoostJob
		rt                  v1.ReplicaType
		index               string
		expectedClusterSpec map[string]string
	}
	testCase := []tc{
		tc{
			job:                 NewXGBoostJobWithMaster(0),
			rt:                  v1alpha1.XGBoostReplicaTypeMaster,
			index:               "0",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "1", "MASTER_PORT": "9999", "RANK": "0", "MASTER_ADDR": "test-xgboostjob-master-0"},
		},
		tc{
			job:                 NewXGBoostJobWithMaster(1),
			rt:                  v1alpha1.XGBoostReplicaTypeMaster,
			index:               "1",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "2", "MASTER_PORT": "9999", "RANK": "1", "MASTER_ADDR": "test-xgboostjob-master-0"},
		},
		tc{
			job:                 NewXGBoostJobWithMaster(2),
			rt:                  v1alpha1.XGBoostReplicaTypeMaster,
			index:               "0",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "9999", "RANK": "0", "MASTER_ADDR": "test-xgboostjob-master-0"},
		},
		tc{
			job:                 NewXGBoostJobWithMaster(2),
			rt:                  v1alpha1.XGBoostReplicaTypeWorker,
			index:               "1",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "9999", "RANK": "1", "MASTER_ADDR": "test-xgboostjob-master-0"},
		},
		tc{
			job:                 NewXGBoostJobWithMaster(2),
			rt:                  v1alpha1.XGBoostReplicaTypeWorker,
			index:               "1",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "9999", "RANK": "1", "MASTER_ADDR": "test-xgboostjob-master-0"},
		},
	}
	for _, c := range testCase {
		demoTemplateSpec := c.job.Spec.XGBReplicaSpecs[v1.ReplicaType(c.rt)].Template
		if err := SetPodEnv(c.job, &demoTemplateSpec, c.index); err != nil {
			t.Errorf("Failed to set cluster spec: %v", err)
		}
		actual := demoTemplateSpec.Spec.Containers[0].Env
		for _, env := range actual {
			if val, ok := c.expectedClusterSpec[env.Name]; ok {
				if val != env.Value {
					t.Errorf("For name %s Got %s. Expected %s ", env.Name, env.Value, c.expectedClusterSpec[env.Name])
				}
			}
		}
	}
}
