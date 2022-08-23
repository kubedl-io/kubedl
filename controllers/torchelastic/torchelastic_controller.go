/*
Copyright 2022 The Alibaba Authors.

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

package torchelastic

import (
	"context"
	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util/concurrent"
	logger "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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
	locker sync.Mutex
	// metrics stores observations collected from running pods
	metrics map[string]map[int32][]MetricObservation
	// torchElasticJobs stores torch-elastic jobs infos.
	torchElasticJobs map[string]TorchElasticJob
}

// TorchElasticJob represents one elastic job.
type TorchElasticJob struct {
	Name       string
	Namespace  string
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// MetricObservation represents one metric set collected from training pods.
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
			delete(ts.metrics, makeElasticJobName(pytorchJob.Name, pytorchJob.Namespace))
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
