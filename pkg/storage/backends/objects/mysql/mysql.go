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

package mysql

import (
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	"github.com/alibaba/kubedl/pkg/storage/backends/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/storage/dmo/converters"
	"github.com/alibaba/kubedl/pkg/util"
)

const (
	// initListSize defines the initial capacity when list objects from backend.
	initListSize = 512
)

func NewMysqlBackendService() backends.ObjectStorageBackend {
	return &mysqlBackend{initialized: 0}
}

var _ backends.ObjectStorageBackend = &mysqlBackend{}

type mysqlBackend struct {
	db          *gorm.DB
	initialized int32
}

func (b *mysqlBackend) Initialize() error {
	if atomic.LoadInt32(&b.initialized) == 1 {
		return nil
	}
	if err := b.init(); err != nil {
		return err
	}
	atomic.StoreInt32(&b.initialized, 1)
	return nil
}

func (b *mysqlBackend) Close() error {
	if b.db == nil {
		return nil
	}
	return b.db.Commit().Close()
}

func (b *mysqlBackend) Name() string {
	return "mysql"
}

func (b *mysqlBackend) SavePod(pod *corev1.Pod, defaultContainerName, region string) error {
	klog.V(5).Infof("[mysql.SavePod] pod: %s/%s", pod.Namespace, pod.Name)

	dmoPod := dmo.Pod{}
	query := &dmo.Pod{PodID: string(pod.UID), Namespace: pod.Namespace, Name: pod.Name}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).First(&dmoPod)
	if result.Error != nil {
		if gorm.IsRecordNotFoundError(result.Error) {
			return b.createNewPod(pod, defaultContainerName, region)
		}
		return result.Error
	}

	newPod, err := converters.ConvertPodToDMOPod(pod, defaultContainerName, region)
	if err != nil {
		return err
	}
	return b.updatePod(&dmoPod, newPod)
}

func (b *mysqlBackend) ListPods(ns, name, jobID string) ([]*dmo.Pod, error) {
	klog.V(5).Infof("[mysql.ListPods] jobID: %s", jobID)

	podList := make([]*dmo.Pod, 0, initListSize)
	query := &dmo.Pod{Namespace: ns, Name: name, JobID: jobID}

	result := b.db.Where(query).
		Order("replica_type").
		Order("CAST(SUBSTRING_INDEX(name, '-', -1) AS SIGNED)").
		Order("gmt_created DESC").
		Find(&podList)
	if result.Error != nil {
		return nil, result.Error
	}
	return podList, nil
}

func (b *mysqlBackend) StopPod(ns, name, podID string) error {
	klog.V(5).Infof("[mysql.StopPod] pod: %s/%s/%s", ns, name, podID)

	oldPod := dmo.Pod{}
	if result := b.db.Where(&dmo.Pod{PodID: podID, Namespace: ns, Name: name}).First(&oldPod); result.Error != nil {
		return result.Error
	}
	if oldPod.Status == utils.PodStopped || oldPod.Status == corev1.PodFailed || oldPod.Status == corev1.PodSucceeded {
		return nil
	}
	newPod := &dmo.Pod{
		Version:     oldPod.Version,
		Status:      oldPod.Status,
		HostIP:      oldPod.HostIP,
		PodIP:       oldPod.PodIP,
		Deleted:     oldPod.Deleted,
		IsInEtcd:    util.IntPtr(0),
		Remark:      oldPod.Remark,
		GmtStarted:  oldPod.GmtStarted,
		GmtFinished: oldPod.GmtFinished,
	}
	if status := oldPod.Status; status == corev1.PodPending || status == corev1.PodRunning || status == corev1.PodUnknown {
		newPod.Status = utils.PodStopped
		newPod.GmtFinished = util.TimePtr(time.Now())
		if newPod.GmtStarted == nil || newPod.GmtStarted.IsZero() {
			newPod.GmtStarted = &oldPod.GmtCreated
		}
	}
	return b.updatePod(&oldPod, newPod)
}

func (b *mysqlBackend) SaveJob(job metav1.Object, kind string, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, jobStatus *apiv1.JobStatus, region string) error {
	klog.V(5).Infof("[mysql.SaveJob] kind: %s job: %s/%s", kind, job.GetNamespace(), job.GetName())

	dmoJob := dmo.Job{}
	query := &dmo.Job{JobID: string(job.GetUID()), Namespace: job.GetNamespace(), Name: job.GetName()}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).First(&dmoJob)
	if result.Error != nil {
		if gorm.IsRecordNotFoundError(result.Error) {
			return b.createNewJob(job, kind, specs, jobStatus, region)
		}
		return result.Error
	}

	newJob, err := converters.ConvertJobToDMOJob(job, kind, specs, jobStatus, region)
	if err != nil {
		return err
	}
	return b.updateJob(&dmoJob, newJob)
}

func (b *mysqlBackend) GetNotebook(ns, name, ID, region string) (*dmo.Notebook, error) {
	klog.V(5).Infof("[mysql.GetNotebook] notebookID: %s", ID)

	notebook := dmo.Notebook{}
	query := &dmo.Notebook{NotebookID: ID, Namespace: ns, Name: name}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).First(&notebook)
	if result.Error != nil {
		return nil, result.Error
	}
	return &notebook, nil
}
func (b *mysqlBackend) GetJob(ns, name, jobID, kind, region string) (*dmo.Job, error) {
	klog.V(5).Infof("[mysql.GetJob] jobID: %s", jobID)

	job := dmo.Job{}
	query := &dmo.Job{JobID: jobID, Namespace: ns, Name: name, Kind: kind}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).First(&job)
	if result.Error != nil {
		return nil, result.Error
	}
	return &job, nil
}

func (b *mysqlBackend) ListJobs(query *backends.Query) ([]*dmo.Job, error) {
	klog.V(5).Infof("[mysql.ListJobs] query: %+v", query)

	jobList := make([]*dmo.Job, 0, initListSize)
	db := b.db.Model(&dmo.Job{})
	db = db.Where("gmt_created < ?", query.EndTime).
		Where("gmt_created > ?", query.StartTime)
	if query.IsDel != nil {
		db = db.Where("deleted = ?", *query.IsDel)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Namespace != "" {
		db = db.Where("namespace LIKE ?", "%"+query.Namespace+"%")
	}
	if query.JobID != "" {
		db = db.Where("job_id = ?", query.JobID)
	}
	if query.Region != "" {
		db = db.Where("deploy_region = ?", query.Region)
	}
	db = db.Order("gmt_created DESC")
	if query.Pagination != nil {
		db = db.Count(&query.Pagination.Count).
			Limit(query.Pagination.PageSize).
			Offset((query.Pagination.PageNum - 1) * query.Pagination.PageSize)
	}
	db = db.Find(&jobList)
	if db.Error != nil {
		return nil, db.Error
	}
	return jobList, nil
}

func (b *mysqlBackend) ListNotebooks(query *backends.NotebookQuery) ([]*dmo.Notebook, error) {
	klog.V(5).Infof("[mysql.ListNotebooks] query: %+v", query)

	notebookList := make([]*dmo.Notebook, 0, initListSize)
	db := b.db.Model(&dmo.Notebook{})
	db = db.Where("gmt_created < ?", query.EndTime).
		Where("gmt_created > ?", query.StartTime)
	if query.IsDel != nil {
		db = db.Where("deleted = ?", *query.IsDel)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Namespace != "" {
		db = db.Where("namespace LIKE ?", "%"+query.Namespace+"%")
	}
	if query.NotebookID != "" {
		db = db.Where("notebook_id = ?", query.NotebookID)
	}
	if query.Region != "" {
		db = db.Where("deploy_region = ?", query.Region)
	}
	db = db.Order("gmt_created DESC")
	if query.Pagination != nil {
		db = db.Count(&query.Pagination.Count).
			Limit(query.Pagination.PageSize).
			Offset((query.Pagination.PageNum - 1) * query.Pagination.PageSize)
	}
	db = db.Find(&notebookList)
	if db.Error != nil {
		return nil, db.Error
	}
	return notebookList, nil
}

func (b *mysqlBackend) DeleteNotebook(ns, name, id, region string) error {
	klog.V(5).Infof("[mysql.DeleteNotebook] notebookID: %s, region: %s", id, region)

	notebook, err := b.GetNotebook(ns, name, id, region)
	if err != nil {
		return err
	}

	if notebook.IsInEtcd != nil && *notebook.IsInEtcd == 1 {
		return errors.New("notebook is still in etcd, must stop it first")
	}
	Deleted := 1
	newNotebook := &dmo.Notebook{
		Namespace:     notebook.Namespace,
		NotebookID:    notebook.NotebookID,
		Version:       notebook.Version,
		Status:        notebook.Status,
		DeployRegion:  notebook.DeployRegion,
		Deleted:       &Deleted,
		IsInEtcd:      notebook.IsInEtcd,
		GmtTerminated: notebook.GmtTerminated,
	}
	return b.updateNotebook(notebook, newNotebook)
}

func (b *mysqlBackend) StopJob(ns, name, jobID, kind, region string) error {
	klog.V(5).Infof("[mysql.StopJob] jobID: %s, region: %s", jobID, region)

	job, err := b.GetJob(ns, name, jobID, kind, region)
	if err != nil {
		return err
	}

	newJob := &dmo.Job{
		JobID:          job.JobID,
		Version:        job.Version,
		Status:         job.Status,
		DeployRegion:   job.DeployRegion,
		Deleted:        job.Deleted,
		IsInEtcd:       util.IntPtr(0),
		GmtJobFinished: job.GmtJobFinished,
	}
	if status := job.Status; status == apiv1.JobRunning || status == apiv1.JobCreated || status == apiv1.JobRestarting {
		newJob.Status = utils.JobStopped
		now := time.Now()
		newJob.GmtJobFinished = util.TimePtr(now)
	}
	return b.updateJob(job, newJob)
}

func (b *mysqlBackend) DeleteJob(ns, name, jobID, kind, region string) error {
	klog.V(5).Infof("[mysql.DeleteJob] jobID: %s, region: %s", jobID, region)

	job, err := b.GetJob(ns, name, jobID, kind, region)
	if err != nil {
		return err
	}

	if job.IsInEtcd != nil && *job.IsInEtcd == 1 {
		return errors.New("job is still in etcd, must stop it first")
	}
	Deleted := 1
	newJob := &dmo.Job{
		Namespace:      job.Namespace,
		JobID:          job.JobID,
		Version:        job.Version,
		Status:         job.Status,
		DeployRegion:   job.DeployRegion,
		Deleted:        &Deleted,
		IsInEtcd:       job.IsInEtcd,
		GmtJobFinished: job.GmtJobFinished,
	}
	return b.updateJob(job, newJob)
}

func (b *mysqlBackend) createNewPod(pod *corev1.Pod, defaultContainerName string, region string) error {
	dmoPod, err := converters.ConvertPodToDMOPod(pod, defaultContainerName, region)
	if err != nil {
		return err
	}
	return b.db.Create(dmoPod).Error
}

func (b *mysqlBackend) updatePod(oldPod, newPod *dmo.Pod) error {
	var (
		oldVersion, newVersion int64
		err                    error
	)
	// Compare versions between two pods.
	if oldVersion, err = strconv.ParseInt(oldPod.Version, 10, 64); err != nil {
		return err
	}
	if newVersion, err = strconv.ParseInt(newPod.Version, 10, 64); err != nil {
		return err
	}
	if oldVersion > newVersion {
		klog.Warningf("try to update a pod newer than the existing one, old version: %d, new version: %d",
			oldVersion, newVersion)
		return nil
	}
	// Setup timestamps if new pod has not set.
	if oldPod.GmtStarted != nil && !oldPod.GmtStarted.IsZero() {
		newPod.GmtStarted = oldPod.GmtStarted
	}
	if oldPod.GmtFinished != nil && !oldPod.GmtFinished.IsZero() {
		newPod.GmtFinished = oldPod.GmtFinished
	}
	// Only update pod when the old one differs with the new one.
	podEquals := oldPod.Version == newPod.Version && oldPod.Status == newPod.Status &&
		(oldPod.GmtStarted != nil && newPod.GmtStarted != nil && oldPod.GmtStarted.Equal(*newPod.GmtStarted)) &&
		(oldPod.GmtFinished != nil && newPod.GmtFinished != nil && oldPod.GmtFinished.Equal(*newPod.GmtFinished)) &&
		(oldPod.IsInEtcd != nil && newPod.IsInEtcd != nil && *oldPod.IsInEtcd == *newPod.IsInEtcd) &&
		(oldPod.Deleted != nil && newPod.Deleted != nil && *oldPod.Deleted == *newPod.Deleted)

	if podEquals {
		return nil
	}

	// Do updating.
	result := b.db.Model(&dmo.Pod{}).Where(&dmo.Pod{
		PodID:        oldPod.PodID,
		Version:      oldPod.Version,
		DeployRegion: oldPod.DeployRegion,
	}).Updates(&dmo.Pod{
		Version:     newPod.Version,
		Status:      newPod.Status,
		Image:       newPod.Image,
		HostIP:      newPod.HostIP,
		PodIP:       newPod.PodIP,
		Deleted:     newPod.Deleted,
		IsInEtcd:    newPod.IsInEtcd,
		Remark:      newPod.Remark,
		GmtCreated:  newPod.GmtCreated,
		GmtStarted:  newPod.GmtStarted,
		GmtFinished: newPod.GmtFinished,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected < 1 {
		klog.Warningf("update pod with no row affected, old version: %s", oldPod.Version)
	}
	return nil
}

func (b *mysqlBackend) createNewJob(job metav1.Object, kind string, specs map[apiv1.ReplicaType]*apiv1.ReplicaSpec, jobStatus *apiv1.JobStatus, region string) error {
	newJob, err := converters.ConvertJobToDMOJob(job, kind, specs, jobStatus, region)
	if err != nil {
		return err
	}
	return b.db.Create(newJob).Error
}

func (b *mysqlBackend) updateNotebook(oldNotebook, newNotebook *dmo.Notebook) error {
	var (
		oldVersion, newVersion int64
		err                    error
	)
	// Compare versions between two pods.
	if oldVersion, err = strconv.ParseInt(oldNotebook.Version, 10, 64); err != nil {
		return err
	}
	if newVersion, err = strconv.ParseInt(newNotebook.Version, 10, 64); err != nil {
		return err
	}
	if oldVersion > newVersion {
		klog.Warningf("try to update a job newer than the existing one, old version: %d, new version: %d",
			oldVersion, newVersion)
		return nil
	}

	// Only update job when the old one differs with the new one.
	equals := oldVersion == newVersion && oldNotebook.Status == newNotebook.Status &&
		(oldNotebook.IsInEtcd != nil && newNotebook.IsInEtcd != nil && *oldNotebook.IsInEtcd == *newNotebook.IsInEtcd) &&
		(oldNotebook.Deleted != nil && newNotebook.Deleted != nil && *oldNotebook.Deleted == *newNotebook.Deleted)
	if equals {
		return nil
	}

	if oldNotebook.GmtRunning != nil && !oldNotebook.GmtRunning.IsZero() && newNotebook.GmtRunning == nil {
		newNotebook.GmtRunning = oldNotebook.GmtRunning
	}

	result := b.db.Model(&dmo.Notebook{}).Where(&dmo.Notebook{
		Name:         oldNotebook.Name,
		NotebookID:   oldNotebook.NotebookID,
		Version:      oldNotebook.Version,
		DeployRegion: oldNotebook.DeployRegion,
	}).Updates(&dmo.Notebook{
		Status:        newNotebook.Status,
		Namespace:     newNotebook.Namespace,
		DeployRegion:  newNotebook.DeployRegion,
		Version:       newNotebook.Version,
		Deleted:       newNotebook.Deleted,
		IsInEtcd:      newNotebook.IsInEtcd,
		GmtCreated:    newNotebook.GmtCreated,
		GmtRunning:    newNotebook.GmtRunning,
		GmtTerminated: newNotebook.GmtTerminated,
	})
	if result.Error != nil {
		return result.Error
	}

	if oldNotebook.Status != string(v1alpha1.NotebookTerminated) {
		if newNotebook.GmtTerminated != nil {
			klog.Infof("[updateJob digest] jobID: %s, duration: %dm, old status: %s, new status: %s",
				newNotebook.NotebookID, newNotebook.GmtTerminated.Sub(oldNotebook.GmtCreated)/time.Minute, oldNotebook.Status, newNotebook.Status)
		} else {
			klog.Infof("[updateJob digest] jobID: %s, old status: %s, new status: %s",
				newNotebook.NotebookID, oldNotebook.Status, newNotebook.Status)
		}
	}
	return nil
}

func (b *mysqlBackend) updateJob(oldJob, newJob *dmo.Job) error {
	var (
		oldVersion, newVersion int64
		err                    error
	)
	// Compare versions between two pods.
	if oldVersion, err = strconv.ParseInt(oldJob.Version, 10, 64); err != nil {
		return err
	}
	if newVersion, err = strconv.ParseInt(newJob.Version, 10, 64); err != nil {
		return err
	}
	if oldVersion > newVersion {
		klog.Warningf("try to update a job newer than the existing one, old version: %d, new version: %d",
			oldVersion, newVersion)
		return nil
	}

	// Only update job when the old one differs with the new one.
	jobEquals := oldVersion == newVersion && oldJob.Status == newJob.Status &&
		(oldJob.IsInEtcd != nil && newJob.IsInEtcd != nil && *oldJob.IsInEtcd == *newJob.IsInEtcd) &&
		(oldJob.Deleted != nil && newJob.Deleted != nil && *oldJob.Deleted == *newJob.Deleted)
	if jobEquals {
		return nil
	}

	if oldJob.GmtJobRunning != nil && !oldJob.GmtJobRunning.IsZero() && newJob.GmtJobRunning == nil {
		newJob.GmtJobRunning = oldJob.GmtJobRunning
	}

	result := b.db.Model(&dmo.Job{}).Where(&dmo.Job{
		Name:         oldJob.Name,
		JobID:        oldJob.JobID,
		Version:      oldJob.Version,
		DeployRegion: oldJob.DeployRegion,
	}).Updates(&dmo.Job{
		Status:         newJob.Status,
		Namespace:      newJob.Namespace,
		DeployRegion:   newJob.DeployRegion,
		Version:        newJob.Version,
		Deleted:        newJob.Deleted,
		IsInEtcd:       newJob.IsInEtcd,
		GmtCreated:     newJob.GmtCreated,
		GmtJobRunning:  newJob.GmtJobRunning,
		GmtJobFinished: newJob.GmtJobFinished,
	})
	if result.Error != nil {
		return result.Error
	}

	if oldJob.Status != apiv1.JobSucceeded && oldJob.Status != apiv1.JobFailed && oldJob.Status != utils.JobStopped {
		if newJob.GmtJobFinished != nil {
			klog.Infof("[updateJob digest] jobID: %s, duration: %dm, old status: %s, new status: %s",
				newJob.JobID, newJob.GmtJobFinished.Sub(oldJob.GmtCreated)/time.Minute, oldJob.Status, newJob.Status)
		} else {
			klog.Infof("[updateJob digest] jobID: %s, old status: %s, new status: %s",
				newJob.JobID, oldJob.Status, newJob.Status)
		}
	}
	return nil
}

func (b *mysqlBackend) CreateWorkspace(workspace *model.WorkspaceInfo) error {
	return nil
}

func (b *mysqlBackend) DeleteWorkspace(name string) error {
	return nil
}

func (b *mysqlBackend) ListWorkspaces(query *backends.WorkspaceQuery) ([]*model.WorkspaceInfo, error) {
	return nil, nil
}

func (b *mysqlBackend) init() error {
	dbSource, logMode, err := GetMysqlDBSource()
	if err != nil {
		return err
	}
	if b.db, err = gorm.Open("mysql", dbSource); err != nil {
		return err
	}
	b.db.LogMode(logMode == "debug")

	// Try create tables if they have not been created in database, or the
	// storage service will not work.
	if !b.db.HasTable(&dmo.Pod{}) {
		klog.Infof("database has not table %s, try to create it", dmo.Pod{}.TableName())
		err = b.db.CreateTable(&dmo.Pod{}).Error
		if err != nil {
			return err
		}
	}
	if !b.db.HasTable(&dmo.Job{}) {
		klog.Infof("database has not table %s, try to create it", dmo.Job{}.TableName())
		err = b.db.CreateTable(&dmo.Job{}).Error
		if err != nil {
			return err
		}
	}
	return nil
}
