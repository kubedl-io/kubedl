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

package v1alpha1

import (
	"testing"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestStorageXGBoostJob(t *testing.T) {
	key := types.NamespacedName{
		Name:      "foo",
		Namespace: "default",
	}
	created := &XGBoostJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: XGBoostJobSpec{
			XGBReplicaSpecs: make(map[v1.ReplicaType]*v1.ReplicaSpec),
		},
		Status: XGBoostJobStatus{
			JobStatus: v1.JobStatus{
				ReplicaStatuses: map[v1.ReplicaType]*v1.ReplicaStatus{},
			},
		},
	}
	g := gomega.NewGomegaWithT(t)

	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	// Test Create
	expected := &XGBoostJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: "xgboostjob.kubeflow.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Spec: XGBoostJobSpec{
			XGBReplicaSpecs: make(map[v1.ReplicaType]*v1.ReplicaSpec),
		},
		Status: XGBoostJobStatus{
			JobStatus: v1.JobStatus{
				ReplicaStatuses: map[v1.ReplicaType]*v1.ReplicaStatus{},
			},
		},
	}
	scheme.Scheme.Default(expected)

	fetched := &XGBoostJob{}
	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(expected).To(gomega.Equal(fetched))

	// Test Updating the Labels
	updated := expected.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, expected)).NotTo(gomega.HaveOccurred())
	g.Expect(expected).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), expected)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, expected)).To(gomega.HaveOccurred())
}
