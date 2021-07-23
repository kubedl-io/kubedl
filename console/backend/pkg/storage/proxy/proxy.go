package proxy

import (
	"github.com/alibaba/kubedl/console/backend/pkg/storage/apiserver"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/backends/objects/mysql"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewProxyBackendService() backends.ObjectStorageBackend {
	return proxyBackend{
		apiServerBackend: apiserver.NewAPIServerBackendService(),
		mysqlBackend:     mysql.NewMysqlBackendService(),
	}
}

type proxyBackend struct {
	apiServerBackend backends.ObjectStorageBackend
	mysqlBackend     backends.ObjectStorageBackend
	hasMysql         bool
}

func (p proxyBackend) Initialize() error {
	if err := p.apiServerBackend.Initialize(); err != nil {
		return err
	}
	if err := p.mysqlBackend.Initialize(); err == nil {
		p.hasMysql = true
	}
	return nil
}

func (p proxyBackend) Close() error {
	if err := p.apiServerBackend.Close(); err != nil {
		return err
	}
	if p.hasMysql {
		if err := p.mysqlBackend.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (p proxyBackend) Name() string {
	return "default"
}

func (p proxyBackend) SavePod(pod *corev1.Pod, defaultContainerName, region string) error {
	return p.apiServerBackend.SavePod(pod, defaultContainerName, region)
}

func (p proxyBackend) ListPods(ns, name, jobID string) ([]*dmo.Pod, error) {
	pods, err := p.apiServerBackend.ListPods(ns, name, jobID)
	if (err == nil && pods != nil) || !p.hasMysql {
		return pods, err
	}
	return p.mysqlBackend.ListPods(ns, name, jobID)
}

func (p proxyBackend) StopPod(ns, name, podID string) error {
	return p.apiServerBackend.StopPod(ns, name, podID)
}

func (p proxyBackend) SaveJob(job metav1.Object, kind string, specs map[v1.ReplicaType]*v1.ReplicaSpec, jobStatus *v1.JobStatus, region string) error {
	return p.apiServerBackend.SaveJob(job, kind, specs, jobStatus, region)
}

func (p proxyBackend) GetJob(ns, name, jobID, kind, region string) (*dmo.Job, error) {
	job, err := p.apiServerBackend.GetJob(ns, name, jobID, kind, region)
	if (err == nil && job != nil) || !p.hasMysql {
		return job, err
	}
	return p.mysqlBackend.GetJob(ns, name, jobID, kind, region)
}

func (p proxyBackend) ListJobs(query *backends.Query) ([]*dmo.Job, error) {
	jobs, err := p.apiServerBackend.ListJobs(query)
	if (err == nil && jobs != nil) || !p.hasMysql {
		return jobs, err
	}
	return p.mysqlBackend.ListJobs(query)
}

func (p proxyBackend) StopJob(ns, name, jobID, kind, region string) error {
	return p.apiServerBackend.StopJob(ns, name, jobID, kind, region)
}

func (p proxyBackend) DeleteJob(ns, name, jobID, kind, region string) error {
	return p.apiServerBackend.DeleteJob(ns, name, jobID, kind, region)
}
