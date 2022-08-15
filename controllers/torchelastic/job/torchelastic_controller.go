package job

import (
	"context"
	"fmt"
	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	controllerv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/concurrent"
	logger "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"sync"
	"time"
)

const (
	controllerName  = "TorchElasticController"
	interval        = 5 * time.Second
	podReadyTimeout = 1 * time.Minute
	logTimeout      = 1 * time.Minute
)

type name string
type namespace string

func init() {
	jobElasticCtrlMap[&training.PyTorchJob{}] = NewTorchElasticController
}

func NewTorchElasticController(mgr ctrl.Manager, period, count int) ElasticController {
	metrics := make(map[string]map[int32][]MetricObservation)
	torchJobs := make(map[string]TorchElasticJob)
	return &TorchElasticController{
		period:           period,
		metricCount:      count,
		client:           kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		metrics:          metrics,
		torchElasticJobs: torchJobs,
		Client:           mgr.GetClient(),
	}
}

type TorchElasticController struct {
	period      int
	metricCount int
	client      *kubernetes.Clientset
	client.Client
	metrics          map[string]map[int32][]MetricObservation
	locker           sync.Mutex
	torchElasticJobs map[string]TorchElasticJob
}

type TorchElasticJob struct {
	Name       string
	Namespace  string
	ctx        context.Context
	cancelFunc context.CancelFunc
}

type MetricObservation struct {
	Epoch    int32   `json:"epoch,omitempty"`
	Batch    int32   `json:"batch,omitempty"`
	Accuracy float64 `json:"accuracy,omitempty"`
	Latency  float64 `json:"latency,omitempty"`
}

func (ts *TorchElasticController) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	pytorchJob := training.PyTorchJob{}
	err := ts.Client.Get(context.Background(), types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, &pytorchJob)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("try to fetch pytorch job but it has been deleted.", "key", req.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (ts *TorchElasticController) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: ts})
	if err != nil {
		return err
	}
	// Watch events with pod events-handler.
	if err = c.Watch(&source.Kind{Type: &training.PyTorchJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: onOwnerCreateFunc(ts),
		DeleteFunc: onOwnerDeleteFunc(ts),
	}); err != nil {
		return err
	}

	ctx := context.Background()
	go wait.UntilWithContext(ctx, ts.startElasticForAllJobs, time.Duration(ts.period)*(time.Second))
	log.Info("Start Elastic Scaling Controller Loop")

	ctx.Done()
	log.Info("Shutting down Elastic Scaling Controller Loop")
	return nil
}

func onOwnerCreateFunc(ts *TorchElasticController) func(e event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		pytorchJob, ok := e.Object.(*training.PyTorchJob)
		if !ok {
			return true
		}
		if !pytorchJob.Spec.EnableElastic && pytorchJob.Spec.ElasticPolicy == nil {
			return true
		}
		ctx, cancel := context.WithCancel(context.Background())
		ctx = context.WithValue(ctx, name("job"), pytorchJob.Name)
		ctx = context.WithValue(ctx, namespace("namespace"), pytorchJob.Namespace)
		logger.Info("Create torch elastic job: ", pytorchJob.Name, " in namespace: ", pytorchJob.Namespace)
		ts.torchElasticJobs[makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace)] = TorchElasticJob{
			Name:       pytorchJob.Name,
			Namespace:  pytorchJob.Namespace,
			ctx:        ctx,
			cancelFunc: cancel,
		}
		return true
	}
}

func onOwnerDeleteFunc(ts *TorchElasticController) func(e event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {
		pytorchJob, ok := e.Object.(*training.PyTorchJob)
		if !ok {
			return true
		}
		if !pytorchJob.Spec.EnableElastic && pytorchJob.Spec.ElasticPolicy == nil {
			return true
		}

		logger.Infof("Deleting elastic scaling for pytorch job %s from namespace %s", pytorchJob.Name, pytorchJob.Namespace)
		// Delete job infos saved in Torch Elastic controller.
		if _, ok := ts.torchElasticJobs[makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace)]; ok {
			cancel := ts.torchElasticJobs[makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace)].cancelFunc
			defer cancel()
			delete(ts.torchElasticJobs, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		}

		return true
	}
}

// Start elastic scaling loop for all torch elastic jobs.
func (ts *TorchElasticController) startElasticForAllJobs(ctx context.Context) {
	tickets := 100 // max semaphore tickets limited.
	if len(ts.torchElasticJobs) < 100 {
		tickets = len(ts.torchElasticJobs)
	}
	sema := concurrent.NewSemaphore(tickets)
	for _, torchJob := range ts.torchElasticJobs {
		sema.Acquire()

		go func(job TorchElasticJob) {
			defer sema.Release()
			//Start elastic scaling for each torch elastic job.
			ts.start(job.ctx, job.cancelFunc, job.Name, job.Namespace)
		}(torchJob)
	}
	// block until all semaphore is released.
	sema.Wait()
}

func (ts *TorchElasticController) start(ctx context.Context, cancel context.CancelFunc, name, namespace string) {
	sharedPytorchJob := &training.PyTorchJob{}
	jobName := name
	jobNamespace := namespace

	// Create metrics for each torch elastic job.
	ts.locker.Lock()
	if _, ok := ts.metrics[jobName]; !ok {
		ts.metrics[jobName] = make(map[int32][]MetricObservation)
	}
	ts.locker.Unlock()

	err := ts.Client.Get(ctx, types.NamespacedName{Namespace: jobNamespace, Name: jobName}, sharedPytorchJob)
	if err != nil {
		logger.Infof("try to get job %s from namespace %s but it has been deleted", jobName, jobNamespace)
		// cancel the elastic scaling process context of the deleted job.
		defer cancel()
		return
	}

	pytorchJob := sharedPytorchJob.DeepCopy()
	if pytorchJob.Spec.ElasticPolicy.MaxReplicas == nil || pytorchJob.Spec.ElasticPolicy.MinReplicas == nil {
		logger.Infof("pytorch job %s does not configure the max or min replicas", pytorchJob.Name)
		defer cancel()
		delete(ts.torchElasticJobs, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		return
	}

	if pytorchJob.Status.ElasticStatus == nil {
		initializeElasticStatuses(&pytorchJob.Status, training.PyTorchReplicaTypeWorker)
		pytorchJob.Status.ElasticStatus[training.PyTorchReplicaTypeWorker].CurrentReplicas = *pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas
		pytorchJob.Status.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue = true
		now := metav1.Now()
		pytorchJob.Status.ElasticStatus[training.PyTorchReplicaTypeWorker].LastUpdateTime = &now
		if err := ts.UpdateJobStatusInApiServer(pytorchJob, &pytorchJob.Status); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				log.Info("fail to update pytorch job")
			}
		}
		return
	}

	jobStatus := pytorchJob.Status.DeepCopy()
	oldStatus := jobStatus.DeepCopy()
	if pytorchJob.Status.CompletionTime != nil || pytorchJob.DeletionTimestamp != nil {
		logger.Infof("job %s has been completed or deleted and does not need to do elastic scaling", pytorchJob.Name)
		defer cancel()
		delete(ts.torchElasticJobs, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		return
	}

	currentReplicas := *pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas
	// Get all pods for the pytorch job.
	pods, err := ts.GetPodsForJob(pytorchJob)
	if err != nil {
		logger.Warnf("Get Pods For Job error %v", err)
	}
	hasPendingPod := false
	hasFailedPod := false

	// Wait for all pods running with timeout seconds.
	waitErr := wait.PollImmediate(interval, podReadyTimeout, func() (bool, error) {
		for _, pod := range pods {
			if isRunning := podRunning(pod); !isRunning {
				return false, nil
			}
		}
		return true, nil
	})
	if waitErr != nil {
		logger.Info("pods did not reach the running state")
	}

	for _, pod := range pods {
		if pod.Status.Phase == v1.PodPending {
			hasPendingPod = true
			break
		}
	}
	for _, pod := range pods {
		if pod.Status.Phase == v1.PodFailed {
			hasFailedPod = true
			break
		}
	}

	// If job has pending pods and current replicas are more than min replicas, return to the last replicas.
	if hasPendingPod && currentReplicas > *pytorchJob.Spec.ElasticPolicy.MinReplicas {
		lastReplicas := jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas
		*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = lastReplicas
		// Return to the last replicas.
		if err := ts.Client.Update(ctx, pytorchJob); err != nil {
			log.Info("fail to update replicas of pytorch job")
		}
		jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue = false
		jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas = jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].CurrentReplicas
		jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].CurrentReplicas = lastReplicas
		jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Message = "there exists pending pods, return to the last replicas"
		now := metav1.Now()
		jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastUpdateTime = &now
		if err := ts.UpdateJobStatusInApiServer(pytorchJob, jobStatus); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				log.Info("fail to update pytorch job")
			}
		}
		return

		// If job has pending pods and current replicas equals to the min replicas, cancel the elastic scaling process context.
	} else if (hasPendingPod && currentReplicas == *pytorchJob.Spec.ElasticPolicy.MinReplicas) || hasFailedPod {
		defer cancel()
		logger.Info("pods did not reach the running state at min replicas or job is failed, so the elastic scaling controller shutdown")
		delete(ts.torchElasticJobs, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
		return
	}

	// If job does not need to be scaled, return directly.
	if !hasPendingPod && jobStatus.ElasticStatus != nil && jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue == false {
		log.Info("pytorch job does not need to be scaled")
		return
	}

	// Read training logs from pytorch pods and save the observation.
	observation, err := read(ts.client, jobNamespace, GetDefaultWorkerName(jobName))
	if err != nil {
		logger.Infof("fail to read training logs: %s", err)
		return
	}

	ts.locker.Lock()
	defer ts.locker.Unlock()

	// Create metrics for current replicas.
	if _, ok := ts.metrics[jobName][currentReplicas]; !ok {
		ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
	}
	ts.metrics[jobName][currentReplicas] = append(ts.metrics[jobName][currentReplicas], observation)
	currentLength := len(ts.metrics[jobName][currentReplicas])
	logger.Infof("Current metric length: %d", currentLength)
	// If current metrics have reached the metric count, judge the next scaling replicas.
	if currentLength >= ts.metricCount {
		if currentReplicas > *pytorchJob.Spec.ElasticPolicy.MinReplicas && currentReplicas <= *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
			lastReplicas := jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas
			currentLatency := ts.metrics[jobName][currentReplicas][currentLength-1].Latency
			lastReplicaLatency := ts.metrics[jobName][lastReplicas][currentLength-1].Latency
			log.Info("last latency: ", lastReplicaLatency, "current latency: ", currentLatency)

			if (lastReplicaLatency / float64(lastReplicas)) > (currentLatency / float64(currentReplicas)) {

				if currentReplicas == *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
					jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Message = "The pytorch job has reached the MaxReplicas"
					jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue = false
					ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
				} else {
					newReplicas := currentReplicas + 1
					*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = newReplicas
					if err := ts.Client.Update(ctx, pytorchJob); err != nil {
						log.Info("fail to update pytorch job")
					}

					jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas = currentReplicas
					jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].CurrentReplicas = newReplicas
					jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Message = "continues to scale pytorch job"
					now := metav1.Now()
					jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastUpdateTime = &now
					jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue = true
					if _, ok := ts.metrics[jobName][newReplicas]; !ok {
						ts.metrics[jobName][newReplicas] = make([]MetricObservation, 0)
					}
				}

			} else {
				*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas
				if err := ts.Client.Update(ctx, pytorchJob); err != nil {
					log.Info("fail to update pytorch job")
				}
				jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].CurrentReplicas = lastReplicas
				jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas = currentReplicas
				jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Message = "The pytorch job does not need to be scaled, then go back to the last choice"
				now := metav1.Now()
				jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastUpdateTime = &now
				jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue = false
				ts.metrics[jobName][lastReplicas] = make([]MetricObservation, 0)
				ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
			}

		} else if currentReplicas == *pytorchJob.Spec.ElasticPolicy.MinReplicas && currentReplicas < *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
			newReplicas := *pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas + 1
			*pytorchJob.Spec.PyTorchReplicaSpecs[training.PyTorchReplicaTypeWorker].Replicas = newReplicas
			if err := ts.Client.Update(ctx, pytorchJob); err != nil {
				log.Info("fail to update pytorch job")
			}

			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].CurrentReplicas = newReplicas
			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastReplicas = currentReplicas
			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Message = "The pytorch job continues to be scaled"
			now := metav1.Now()
			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].LastUpdateTime = &now
			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue = true

			if _, ok := ts.metrics[jobName][newReplicas]; !ok {
				ts.metrics[jobName][newReplicas] = make([]MetricObservation, 0)
			}

		} else if currentReplicas == *pytorchJob.Spec.ElasticPolicy.MaxReplicas {
			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Message = "The pytorch job has reached the MaxReplicas"
			jobStatus.ElasticStatus[training.PyTorchReplicaTypeWorker].Continue = false
			ts.metrics[jobName][currentReplicas] = make([]MetricObservation, 0)
		}
	}

	// No need to update the job status if the status hasn't changed since last time.
	if !reflect.DeepEqual(*oldStatus, jobStatus) {
		if err = ts.UpdateJobStatusInApiServer(pytorchJob, jobStatus); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				logger.Info("fail to update pytorch job status")
				return
			}
		}
	}

	return
}

func (ts *TorchElasticController) GetPodsForJob(job *training.PyTorchJob) ([]*v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: ts.GenLabels(job.Name),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podList := &v1.PodList{}
	err = ts.Client.List(context.Background(), podList, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	return commonutil.ToPodPointerList(podList.Items), nil
}

func (ts *TorchElasticController) GenLabels(jobName string) map[string]string {

	labelGroupName := apiv1.GroupNameLabel
	labelJobName := apiv1.JobNameLabel
	groupName := ts.GetGroupNameLabelValue()
	return map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(jobName, "/", "-", -1),
	}
}

func (ts *TorchElasticController) GetGroupNameLabelValue() string {
	return training.SchemeGroupVersion.Group
}

func (ts *TorchElasticController) ControllerName() string {
	return controllerName
}

func makeElasticJobName(name, namespace string) string {
	return name + "-" + namespace
}

// UpdateJobStatusInApiServer updates the job status in API server
func (ts *TorchElasticController) UpdateJobStatusInApiServer(job interface{}, jobStatus *controllerv1.JobStatus) error {
	torchElasticJob, ok := job.(*training.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PytorchJob", torchElasticJob)
	}
	var jobCpy *training.PyTorchJob
	// Job status passed in differs with status in job, update in basis of the passed in one.
	jobCpy = torchElasticJob.DeepCopy()
	jobCpy.Status = *jobStatus.DeepCopy()
	return ts.Status().Update(context.Background(), jobCpy)
}

// initializeReplicaStatuses initializes the ElasticStatuses for replica.
func initializeElasticStatuses(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType) {
	if jobStatus.ElasticStatus == nil {
		jobStatus.ElasticStatus = make(map[apiv1.ReplicaType]*apiv1.ElasticScalingStatus)
	}

	jobStatus.ElasticStatus[rtype] = &apiv1.ElasticScalingStatus{}
}
