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

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/storage/dmo/converters"
	"github.com/alibaba/kubedl/pkg/util"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	// initListSize defines the initial capacity when list objects from backend.
	initListSize = 512

	// Expanded object statuses only used in persistent layer.
	PodStopped = "Stopped"
	JobStopped = "Stopped"
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

func (b *mysqlBackend) ListPods(jobID, region string) ([]*dmo.Pod, error) {
	klog.V(5).Infof("[mysql.ListPods] jobID: %s", jobID)

	podList := make([]*dmo.Pod, 0, initListSize)
	query := &dmo.Pod{JobID: jobID}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).
		Order("replica_type").
		Order("CAST(SUBSTRING_INDEX(name, '-', -1) AS SIGNED)").
		Order("gmt_create DESC").
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
		newPod.Status = PodStopped
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

func (b *mysqlBackend) GetJob(ns, name, jobID, region string) (*dmo.Job, error) {
	klog.V(5).Infof("[mysql.GetJob] jobID: %s", jobID)

	job := dmo.Job{}
	query := &dmo.Job{JobID: jobID, Namespace: ns, Name: name}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).First(&job)
	if result.Error != nil {
		return nil, result.Error
	}
	return &job, nil
}

func (b *mysqlBackend) ListJobs(query backends.Query) ([]*dmo.Job, error) {
	klog.V(5).Infof("[mysql.ListJobs] query: %+v", query)

	jobList := make([]*dmo.Job, 0, initListSize)
	db := b.db.Model(&dmo.Job{})
	db = db.Where("gmt_create < ?", query.EndTime).
		Where("gmt_create > ?", query.StartTime)
	if query.IsDel != nil {
		db = db.Where("is_del = ?", *query.IsDel)
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
	db = db.Order("gmt_submit DESC")
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

func (b *mysqlBackend) StopJob(ns, name, jobID, region string) error {
	klog.V(5).Infof("[mysql.StopJob] jobID: %s, region: %s", jobID, region)

	job := dmo.Job{}
	query := &dmo.Job{JobID: jobID, Namespace: ns, Name: name}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).First(&job)
	if result.Error != nil {
		return result.Error
	}

	newJob := &dmo.Job{
		JobID:        job.JobID,
		Version:      job.Version,
		Status:       job.Status,
		DeployRegion: job.DeployRegion,
		Deleted:      job.Deleted,
		IsInEtcd:     util.IntPtr(0),
		GmtFinished:  job.GmtFinished,
	}
	if status := job.Status; status == apiv1.JobRunning || status == apiv1.JobCreated || status == apiv1.JobRestarting {
		newJob.Status = JobStopped
		newJob.GmtFinished = util.TimePtr(time.Now())
	}
	return b.updateJob(&job, newJob)
}

func (b *mysqlBackend) DeleteJob(ns, name, jobID, region string) error {
	klog.V(5).Infof("[mysql.DeleteJob] jobID: %s, region: %s", jobID, region)

	job := dmo.Job{}
	query := &dmo.Job{JobID: jobID, Namespace: ns, Name: name}
	if region != "" {
		query.DeployRegion = &region
	}
	result := b.db.Where(query).First(&job)
	if result.Error != nil {
		return result.Error
	}
	if job.IsInEtcd != nil && *job.IsInEtcd == 1 {
		return errors.New("job is still in etcd, must stop it first")
	}
	isDel := 1
	newJob := &dmo.Job{
		Namespace:    job.Namespace,
		JobID:        job.JobID,
		Version:      job.Version,
		Status:       job.Status,
		DeployRegion: job.DeployRegion,
		Deleted:      &isDel,
		IsInEtcd:     job.IsInEtcd,
		GmtFinished:  job.GmtFinished,
	}
	return b.updateJob(&job, newJob)
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

	result := b.db.Model(&dmo.Job{}).Where(&dmo.Job{
		Name:         oldJob.Name,
		JobID:        oldJob.JobID,
		Version:      oldJob.Version,
		DeployRegion: oldJob.DeployRegion,
	}).Updates(&dmo.Job{
		Status:       newJob.Status,
		Namespace:    newJob.Namespace,
		DeployRegion: newJob.DeployRegion,
		Version:      newJob.Version,
		Deleted:      newJob.Deleted,
		IsInEtcd:     newJob.IsInEtcd,
		GmtFinished:  newJob.GmtFinished,
	})
	if result.Error != nil {
		return result.Error
	}

	if oldJob.Status != apiv1.JobSucceeded && oldJob.Status != apiv1.JobFailed && oldJob.Status != JobStopped {
		if newJob.GmtFinished != nil {
			klog.Infof("[updateJob digest] jobID: %s, duration: %dm, old status: %s, new status: %s",
				newJob.JobID, newJob.GmtFinished.Sub(oldJob.GmtCreated)/time.Minute, oldJob.Status, newJob.Status)
		} else {
			klog.Infof("[updateJob digest] jobID: %s, old status: %s, new status: %s",
				newJob.JobID, oldJob.Status, newJob.Status)
		}
	}
	return nil
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
