// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package job_controller

import (
	"fmt"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	errPortNotFound = fmt.Errorf("failed to found the port")
)

func GenGeneralName(jobName, rtype, index string) string {
	n := jobName + "-" + rtype + "-" + index
	return strings.Replace(n, "/", "-", -1)
}

// RecheckDeletionTimestamp returns a CanAdopt() function to recheck deletion.
//
// The CanAdopt() function calls getObject() to fetch the latest value,
// and denies adoption attempts if that object has a non-nil DeletionTimestamp.
func RecheckDeletionTimestamp(getObject func() (metav1.Object, error)) func() error {
	return func() error {
		obj, err := getObject()
		if err != nil {
			return fmt.Errorf("can't recheck DeletionTimestamp: %v", err)
		}
		if obj.GetDeletionTimestamp() != nil {
			return fmt.Errorf("%v/%v has just been deleted at %v", obj.GetNamespace(), obj.GetName(), obj.GetDeletionTimestamp())
		}
		return nil
	}
}

func GenExpectationPodsKey(jobKey, replicaType string) string {
	return jobKey + "/" + strings.ToLower(replicaType) + "/pods"
}

func GenExpectationServicesKey(jobKey, replicaType string) string {
	return jobKey + "/" + strings.ToLower(replicaType) + "/services"
}

// GetPortFromJob gets the port of job default container.
func GetPortFromJob(spec map[v1.ReplicaType]*v1.ReplicaSpec, rtype v1.ReplicaType, containerName, portName string) (int32, error) {
	containers := spec[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == containerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == portName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, errPortNotFound
}
