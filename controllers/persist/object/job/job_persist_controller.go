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

package job

import (
	stderrors "errors"
	"fmt"
	"github.com/alibaba/kubedl/pkg/storage/backends/registry"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/util/workloadgate"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// PersistController implementations
type PersistController interface {
	SetupWithManager(mgr ctrl.Manager) error
}

type newJobPersistController func(mgr ctrl.Manager, handler *jobPersistHandler) PersistController

var (
	log               = logf.Log.WithName("job-persist-controller")
	jobPersistCtrlMap = make(map[runtime.Object]newJobPersistController)
)

func NewJobPersistControllers(mgr ctrl.Manager, objStorage string, region string) (PersistController, error) {
	if objStorage == "" {
		return nil, stderrors.New("empty object storage backend name")
	}

	// Get object storage backend from backends registry.
	jobBackend := registry.GetObjectBackend(objStorage)
	if jobBackend == nil {
		return nil, fmt.Errorf("object storage backend [%s] has not registered", objStorage)
	}

	// Initialize obj storage backend before job-persist-controller created.
	if err := jobBackend.Initialize(); err != nil {
		return nil, err
	}

	handler := &jobPersistHandler{region: region, jobBackend: jobBackend}
	pc := jobPersistController{subControllers: make([]PersistController, 0)}

	// Init sub persist controllers for those installed CRD workloads.
	for obj, newCtrl := range jobPersistCtrlMap {
		if _, enabled := workloadgate.IsWorkloadEnable(obj, mgr.GetScheme()); enabled {
			pc.addNewJobPersistController(newCtrl(mgr, handler))
		}
	}

	return &pc, nil
}

var _ PersistController = &jobPersistController{}

type jobPersistController struct {
	subControllers []PersistController
}

func (pc *jobPersistController) SetupWithManager(mgr ctrl.Manager) error {
	for i := range pc.subControllers {
		if err := pc.subControllers[i].SetupWithManager(mgr); err != nil {
			return err
		}
	}
	return nil
}

func (pc *jobPersistController) addNewJobPersistController(c PersistController) {
	pc.subControllers = append(pc.subControllers, c)
}

type jobPersistHandler struct {
	region     string
	jobBackend backends.ObjectStorageBackend
}

func (h *jobPersistHandler) Delete(ns, name, jobID string) error {
	err := h.jobBackend.StopJob(ns, name, jobID, h.region)
	if err != nil {
		log.Error(err, "failed to stop job in object storage backend", "backend name",
			h.jobBackend.Name(), "namespace", ns, "name", name, "job id", jobID)
		return err
	}

	err = h.jobBackend.DeleteJob(ns, name, jobID, h.region)
	if err != nil {
		log.Error(err, "failed to delete job in object storage backend", "backend name",
			h.jobBackend.Name(), "namespace", ns, "name", name, "job id", jobID)
		return err
	}
	return nil
}

func (h *jobPersistHandler) Save(job metav1.Object, kind string, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, jobStatus *apiv1.JobStatus) error {
	err := h.jobBackend.SaveJob(job, kind, specs, jobStatus, h.region)
	if err != nil {
		log.Error(err, "failed to save job in object storage backend", "backend name", h.jobBackend.Name(), "job kind", kind, "job id", job.GetUID())
		return err
	}
	return nil
}
