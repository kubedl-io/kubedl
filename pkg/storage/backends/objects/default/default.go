package defaultBackend

import (
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/backends/objects/apiserver"
	"github.com/alibaba/kubedl/pkg/storage/backends/objects/mysql"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDefaultBackendService() backends.ObjectStorageBackend {
	return defaultBackend{
		apiServerBackend: apiserver.NewAPIServerBackendService(),
		mysqlBackend:     mysql.NewMysqlBackendService(),
	}
}

type defaultBackend struct {
	apiServerBackend backends.ObjectStorageBackend
	mysqlBackend     backends.ObjectStorageBackend
	hasMysql         bool
}

func (d defaultBackend) Initialize() error {
	if err := d.apiServerBackend.Initialize(); err != nil {
		return err
	}
	if err := d.mysqlBackend.Initialize(); err == nil {
		d.hasMysql = true
	}
	return nil
}

func (d defaultBackend) Close() error {
	if err := d.apiServerBackend.Close(); err != nil {
		return err
	}
	if d.hasMysql {
		if err := d.mysqlBackend.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (d defaultBackend) Name() string {
	return "default"
}

func (d defaultBackend) SavePod(pod *corev1.Pod, defaultContainerName, region string) error {
	return d.apiServerBackend.SavePod(pod, defaultContainerName, region)
}

func (d defaultBackend) ListPods(ns, name, jobID string) ([]*dmo.Pod, error) {
	pods, err := d.apiServerBackend.ListPods(ns, name, jobID)
	if err == nil || !d.hasMysql {
		return pods, err
	}
	return d.mysqlBackend.ListPods(ns, name, jobID)
}

func (d defaultBackend) StopPod(ns, name, podID string) error {
	return d.apiServerBackend.StopPod(ns, name, podID)
}

func (d defaultBackend) SaveJob(job metav1.Object, kind string, specs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, region string) error {
	return d.apiServerBackend.SaveJob(job, kind, specs, jobStatus, region)
}

func (d defaultBackend) GetJob(ns, name, jobID, kind, region string) (*dmo.Job, error) {
	job, err := d.apiServerBackend.GetJob(ns, name, jobID, kind, region)
	if err == nil || !d.hasMysql {
		return job, err
	}
	return d.mysqlBackend.GetJob(ns, name, jobID, kind, region)
}

func (d defaultBackend) ListJobs(query *backends.Query) ([]*dmo.Job, error) {
	jobs, err := d.apiServerBackend.ListJobs(query)
	if err == nil || !d.hasMysql {
		return jobs, err
	}
	return d.mysqlBackend.ListJobs(query)
}

func (d defaultBackend) StopJob(ns, name, jobID, kind, region string) error {
	return d.apiServerBackend.StopJob(ns, name, jobID, kind, region)
}

func (d defaultBackend) DeleteJob(ns, name, jobID, kind, region string) error {
	return d.apiServerBackend.DeleteJob(ns, name, jobID, kind, region)
}
