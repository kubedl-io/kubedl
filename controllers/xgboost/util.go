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
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"os"
)

func createClientSets(config *restclientset.Config) (kubeclientset.Interface, kubeclientset.Interface, kubebatchclient.Interface, error) {

	if config == nil {
		println("there is an error for the input config")
		return nil, nil, nil, nil
	}

	kubeClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "xgboostjob-operator"))
	if err != nil {
		return nil, nil, nil, err
	}

	leaderElectionClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "leader-election"))
	if err != nil {
		return nil, nil, nil, err
	}

	kubeBatchClientSet, err := kubebatchclient.NewForConfig(restclientset.AddUserAgent(config, "kube-batch"))
	if err != nil {
		return nil, nil, nil, err
	}

	return kubeClientSet, leaderElectionClientSet, kubeBatchClientSet, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func isGangSchedulerSet(replicas map[v1.ReplicaType]*v1.ReplicaSpec) bool {
	for _, spec := range replicas {
		if spec.Template.Spec.SchedulerName == gangSchedulerName {
			return true
		}
	}
	return false
}
