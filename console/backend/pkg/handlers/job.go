package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/types"

	clientmgr "github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	consoleutils "github.com/alibaba/kubedl/console/backend/pkg/utils"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/backends/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultUser = "Anonymous"
)

func NewJobHandler(objBackend backends.ObjectStorageBackend, logHandler *LogHandler) *JobHandler {
	return &JobHandler{
		logHandler:    logHandler,
		objectBackend: objBackend,
		client:        clientmgr.GetCtrlClient(),
		preSubmitHooks: []preSubmitHook{
			tfJobPreSubmitAutoConvertReplicas,
			pytorchJobPreSubmitAutoConvertReplicas,
			tfJobPreSubmitTensorBoardDefaults,
			pytorchJobPreSubmitTensorBoardDefaults,
		},
	}
}

type JobHandler struct {
	client         client.Client
	logHandler     *LogHandler
	objectBackend  backends.ObjectStorageBackend
	preSubmitHooks []preSubmitHook
}

func (jh *JobHandler) ListJobsFromBackend(query *backends.Query) ([]model.JobInfo, error) {
	jobs, err := jh.objectBackend.ListJobs(query)
	if err != nil {
		return nil, err
	}
	jobInfos := make([]model.JobInfo, 0, len(jobs))
	for _, job := range jobs {
		jobInfos = append(jobInfos, model.ConvertDMOJobToJobInfo(job))
	}
	return jobInfos, nil
}

func (jh *JobHandler) GetDetailedJobFromBackend(ns, name, jobID, kind, region string) (model.JobInfo, error) {
	job, err := jh.objectBackend.GetJob(ns, name, jobID, kind, region)
	if err != nil {
		klog.Errorf("failed to get job from backend, err: %v", err)
		return model.JobInfo{}, err
	}
	pods, err := jh.objectBackend.ListPods(ns, name, jobID)
	if err != nil {
		klog.Errorf("failed to list pods from backend, err: %v", err)
		return model.JobInfo{}, err
	}

	var (
		specs               = make([]model.Spec, 0, len(pods))
		specReplicaStatuses = make(map[string]*model.SpecReplicaStatus)
	)

	for _, pod := range pods {
		if _, ok := specReplicaStatuses[pod.ReplicaType]; !ok {
			specReplicaStatuses[pod.ReplicaType] = &model.SpecReplicaStatus{}
		}

		switch pod.Status {
		case corev1.PodSucceeded:
			specReplicaStatuses[pod.ReplicaType].Succeeded++
		case corev1.PodFailed:
			specReplicaStatuses[pod.ReplicaType].Failed++
		case utils.PodStopped:
			specReplicaStatuses[pod.ReplicaType].Stopped++
		default:
			specReplicaStatuses[pod.ReplicaType].Active++
		}

		specs = append(specs, model.ConvertDMOPodToJobSpec(pod))
	}

	jobInfo := model.ConvertDMOJobToJobInfo(job)
	jobInfo.Specs = specs
	jobInfo.SpecsReplicaStatuses = specReplicaStatuses

	return jobInfo, nil
}

func (jh *JobHandler) StopJobFromBackend(ns, name, jobID, kind, region string) error {
	return jh.objectBackend.StopJob(ns, name, jobID, kind, region)
}

func (jh *JobHandler) DeleteJobFromBackend(ns, name, jobID, kind, region string) error {
	return jh.objectBackend.DeleteJob(ns, name, jobID, kind, region)
}

func (jh *JobHandler) GetJobYamlData(ns, name, kind string) ([]byte, error) {
	job := consoleutils.InitJobRuntimeObjectByKind(kind)
	if job == nil {
		return nil, fmt.Errorf("unsupported job kind: %s", kind)
	}

	err := jh.client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, job)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(job)
}

func (jh *JobHandler) GetJobJsonData(ns, name, kind string) ([]byte, error) {
	job := consoleutils.InitJobRuntimeObjectByKind(kind)
	if job == nil {
		return nil, fmt.Errorf("unsupported job kind: %s", kind)
	}

	err := jh.client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, job)

	if err != nil {
		return nil, err
	}

	return json.Marshal(job)
}

func (jh *JobHandler) SubmitJobWithKind(data []byte, kind string) error {
	job := consoleutils.InitJobRuntimeObjectByKind(kind)

	err := json.Unmarshal(data, job)
	if err == nil {
		return jh.submitJob(job)
	}

	err = yaml.Unmarshal(data, job)
	if err != nil {
		klog.Errorf("failed to unmarshal %s job in yaml format, fallback to json marshalling then, data: %s", kind, string(data))
		return err
	}
	return jh.submitJob(job)
}

func (jh *JobHandler) submitJob(job client.Object) error {
	for _, hook := range jh.preSubmitHooks {
		hook(job)
	}

	return jh.client.Create(context.Background(), job)
}

func (jh *JobHandler) ListPVC(ns string) ([]string, error) {
	list := &corev1.PersistentVolumeClaimList{}
	if err := jh.client.List(context.TODO(), list, &client.ListOptions{Namespace: ns}); err != nil {
		return nil, err
	}
	ret := make([]string, 0, len(list.Items))
	for _, pvc := range list.Items {
		ret = append(ret, pvc.Name)
	}
	return ret, nil
}

func (jh *JobHandler) GetJobStatisticsFromBackend(query *backends.Query) (model.JobStatistics, error) {
	jobStatistics := model.JobStatistics{}
	jobInfos, err := jh.ListJobsFromBackend(query)
	if err != nil {
		return jobStatistics, err
	}

	historyJobsMap := make(map[string]*model.HistoryJobStatistic)
	totalJobCount := int32(0)
	for _, jobInfo := range jobInfos {
		userID := jobInfo.JobUserName
		if len(userID) == 0 {
			userID = defaultUser
		}

		if _, ok := historyJobsMap[userID]; !ok {
			historyJobsMap[userID] = &model.HistoryJobStatistic{}
		}
		historyJobsMap[userID].UserName = userID
		historyJobsMap[userID].JobCount++
		totalJobCount++
	}

	jobStatistics.TotalJobCount = totalJobCount
	for _, stat := range historyJobsMap {
		ratio := float64(stat.JobCount*100) / float64(totalJobCount)
		stat.JobRatio, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", ratio), 64)
		jobStatistics.HistoryJobs = append(jobStatistics.HistoryJobs, stat)
	}

	// Sort history jobs by job ratio
	sort.SliceStable(jobStatistics.HistoryJobs, func(i, j int) bool {
		return jobStatistics.HistoryJobs[i].JobRatio > jobStatistics.HistoryJobs[j].JobRatio
	})

	jobStatistics.StartTime = query.StartTime.Format(time.RFC3339)
	jobStatistics.EndTime = query.EndTime.Format(time.RFC3339)

	return jobStatistics, nil
}

func (jh *JobHandler) GetRunningJobsFromBackend(query *backends.Query) ([]model.JobInfo, error) {
	runningJobs, err := jh.ListJobsFromBackend(query)
	if err != nil {
		return runningJobs, err
	}

	/*
		// sort by job resource
		sort.SliceStable(runningJobs, func(i, j int) bool {
			return runningJobs[i].JobResource.TotalGPU > runningJobs[j].JobResource.TotalGPU ||
				runningJobs[i].JobResource.TotalCPU > runningJobs[j].JobResource.TotalCPU ||
				runningJobs[i].JobResource.TotalMemory > runningJobs[j].JobResource.TotalMemory
		})
	*/

	return runningJobs, nil
}
