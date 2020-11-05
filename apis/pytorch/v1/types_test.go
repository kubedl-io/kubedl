/*
Copyright 2019 The Alibaba Authors.

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

package v1

import (
	"context"
	"testing"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestStoragePytorchJob(t *testing.T) {
	key := types.NamespacedName{
		Name:      "foo",
		Namespace: "default",
	}
	created := &PyTorchJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: PyTorchJobSpec{
			PyTorchReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{},
		},
		Status: v1.JobStatus{
			ReplicaStatuses: map[v1.ReplicaType]*v1.ReplicaStatus{},
			Conditions:      []v1.JobCondition{},
		},
	}
	g := gomega.NewGomegaWithT(t)

	expected := &PyTorchJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: GroupName + "/" + GroupVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Spec: PyTorchJobSpec{
			PyTorchReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{},
		},
		Status: v1.JobStatus{
			ReplicaStatuses: map[v1.ReplicaType]*v1.ReplicaStatus{},
			Conditions:      []v1.JobCondition{},
		},
	}
	scheme.Scheme.Default(expected)

	// Test Create
	fetched := &PyTorchJob{}
	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(expected))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.HaveOccurred())
}
