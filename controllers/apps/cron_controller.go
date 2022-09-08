/*
Copyright 2021 The Alibaba Authors.

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

package cron

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	cronutil "github.com/robfig/cron/v3"

	"github.com/alibaba/kubedl/apis/apps/v1alpha1"
	inferencev1alpha1 "github.com/alibaba/kubedl/apis/inference/v1alpha1"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	utilruntime "github.com/alibaba/kubedl/pkg/util/runtime"
	"github.com/alibaba/kubedl/pkg/util/workloadgate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const cronControllerName = "CronController"

func NewCronController(mgr ctrl.Manager, _ options.JobControllerConfiguration) *CronController {
	return &CronController{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor(cronControllerName),
		extCodec:  utilruntime.NewRawExtensionCodec(mgr.GetScheme()),
	}
}

// CronController reconciles a Workload object.
type CronController struct {
	client    client.Client
	apiReader client.Reader
	recorder  record.EventRecorder
	scheme    *runtime.Scheme
	extCodec  *utilruntime.RawExtensionCodec
}

func (cc *CronController) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(cronControllerName, mgr, controller.Options{
		Reconciler:              cc,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &v1alpha1.Cron{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Discover installed workload and watch for Cron controlled objects.
	for _, workload := range workloads {
		if _, enabled := workloadgate.IsWorkloadEnable(workload, cc.scheme); enabled {
			if err = c.Watch(&source.Kind{Type: workload}, &handler.EnqueueRequestForOwner{
				OwnerType:    &v1alpha1.Cron{},
				IsController: true,
			}, predicate.Funcs{
				CreateFunc: onCronWorkloadCreate,
				DeleteFunc: onCronWorkloadDelete,
				UpdateFunc: onCronWorkloadUpdate,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Reconcile reconciles a cron object.
// +kubebuilder:rbac:groups=apps.kubedl.io,resources=crons,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kubedl.io,resources=crons/status,verbs=get;update;patch

func (cc *CronController) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(3).Infof("start to reconcile cron: %v", req.String())

	cron := v1alpha1.Cron{}
	err := cc.client.Get(context.Background(), req.NamespacedName, &cron)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("cron %v not found, it may has been deleted.", req.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	actives, err := cc.listActiveWorkloads(&cron)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err = cc.refreshCronHistory(&cron, actives); err != nil {
		return ctrl.Result{}, err
	}

	nextDuration, err := cc.syncCron(&cron, actives)
	if err != nil {
		return ctrl.Result{}, err
	}
	if nextDuration != nil {
		return ctrl.Result{RequeueAfter: *nextDuration}, nil
	}
	return ctrl.Result{}, nil
}

func (cc *CronController) syncCron(cron *v1alpha1.Cron, workloads []metav1.Object) (nextDuration *time.Duration, err error) {
	cron = cron.DeepCopy()
	now := time.Now()

	// 1) trim finished and missed workload from active list.
	err = cc.trimFinishedWorkloadsFromActiveList(cron, workloads)
	if err != nil {
		return nil, err
	}
	// 2) apply latest cron status to cluster.
	if err = cc.client.Status().Update(context.Background(), cron); err != nil {
		klog.Errorf("unable to update status for cron %s/%s, err: %v", cron.Namespace, cron.Name, err)
		return nil, err
	}
	// 3) check deletion/suspend/deadline state from retrieved cron object.
	if cron.DeletionTimestamp != nil {
		klog.V(3).Infof("cron %s/%s has been deleted at %v.", cron.Namespace, cron.Name, cron.DeletionTimestamp)
		return nil, nil
	}
	if cron.Spec.Suspend != nil && *cron.Spec.Suspend {
		klog.V(3).Infof("cron %s/%s has been suspended.", cron.Namespace, cron.Name)
		return nil, nil
	}
	if cron.Spec.Deadline != nil && now.After(cron.Spec.Deadline.Time) {
		klog.V(2).Infof("cron %s/%s has reached deadline and will not trigger scheduling anymore.", cron.Namespace, cron.Name)
		cc.recorder.Event(cron, corev1.EventTypeNormal, "Deadline", "cron has reach deadline and stop scheduling")
		return nil, nil
	}
	// 4) schedule next workload if schedule time has come.
	nextDuration, err = cc.scheduleNextIfPossible(cron, now)
	if err != nil {
		klog.V(2).Infof("error scheduling next workload, err: %v", err)
		return nil, err
	}
	return nextDuration, nil
}

func (cc *CronController) scheduleNextIfPossible(cron *v1alpha1.Cron, now time.Time) (*time.Duration, error) {
	sched, err := cronutil.ParseStandard(cron.Spec.Schedule)
	if err != nil {
		// this is likely a user error in defining the spec value
		// we should log the error and not reconcile this cronjob until an update to spec.
		klog.Warningf("invalid schedule in cron: %s", cron.Spec.Schedule)
		cc.recorder.Eventf(cron, corev1.EventTypeWarning, "InvalidSchedule", "invalid schedule: %s, err: %v",
			cron.Spec.Schedule, err)
		return nil, nil
	}
	scheduledTime, err := getNextScheduleTime(cron, now, sched, cc.recorder)
	if err != nil {
		// this is likely a user error in defining the spec value
		// we should log the error and not reconcile this cronjob until an update to spec
		msg := fmt.Sprintf("invalid schedule,  schedule: %s, err: %v", cron.Spec.Schedule, err)
		klog.Warningf(msg)
		cc.recorder.Event(cron, corev1.EventTypeWarning, "InvalidSchedule", msg)
		return nil, err
	}
	if scheduledTime == nil {
		klog.V(2).Info("No unmet start time")
		return nextScheduledTimeDuration(sched, now), nil
	}
	if cron.Status.LastScheduleTime.Equal(&metav1.Time{Time: *scheduledTime}) {
		klog.V(4).Infof("Not starting because the scheduled time is already precessed, cron: %s/%s", cron.Namespace, cron.Name)
		return nextScheduledTimeDuration(sched, now), nil
	}
	if cron.Spec.ConcurrencyPolicy == v1.ForbidConcurrent && len(cron.Status.Active) > 0 {
		// Regardless which source of information we use for the set of active jobs,
		// there is some risk that we won't see an active job when there is one.
		// (because we haven't seen the status update to the Cron or the created pod).
		// So it is theoretically possible to have concurrency with Forbid.
		// As long the as the invocations are "far enough apart in time", this usually won't happen.
		//
		// TODO: for Forbid, we could use the same name for every execution, as a lock.
		klog.V(4).Infof("Not starting because prior execution is still running and concurrency policy is Forbid, cron: %s/%s", cron.Namespace, cron.Name)
		cc.recorder.Eventf(cron, corev1.EventTypeNormal, "AlreadyActive", "Not starting because prior execution is running and concurrency policy is Forbid")
		return nextScheduledTimeDuration(sched, now), nil
	}
	if cron.Spec.ConcurrencyPolicy == v1.ReplaceConcurrent {
		for _, active := range cron.Status.Active {
			klog.V(4).Infof("Deleting workload that was still running at next scheduled start time, workload: %s/%s", active.Namespace, active.Name)

			if err = cc.deleteWorkload(cron, active); err != nil {
				return nil, err
			}
		}
	}

	workloadToCreate, err := cc.newWorkloadFromTemplate(cron, *scheduledTime)
	if err != nil {
		klog.Errorf("unable to initialize workload from cron template, err: %v", err)
		return nil, err
	}
	err = cc.client.Create(context.Background(), workloadToCreate)
	switch {
	case errors.IsAlreadyExists(err):
		// If workload is created by other actor and has already in active list, assume it has updated the cron
		// status accordingly, else not return and fallback to append active list then update status.
		klog.Infof("Workload already exists, cron: %s/%s, workload: %v", cron.Namespace, cron.Name, workloadToCreate)
		if metaWorkload, ok := workloadToCreate.(metav1.Object); ok && inActiveList(cron.Status.Active, metaWorkload) {
			return nil, err
		}
	case err != nil:
		cc.recorder.Eventf(cron, corev1.EventTypeWarning, "FailedCreate", "Error creating workload: %v", err)
		return nil, err
	}

	ref, err := reference.GetReference(cc.scheme, workloadToCreate)
	if err != nil {
		return nil, err
	}
	cc.recorder.Eventf(cron, corev1.EventTypeNormal, "SuccessfulCreate", "Created workload: %v", ref.Name)

	cron.Status.Active = append(cron.Status.Active, *ref)
	cron.Status.LastScheduleTime = &metav1.Time{Time: *scheduledTime}
	if err = cc.client.Status().Update(context.Background(), cron); err != nil {
		klog.Errorf("unable to update status, cron: %s/%s, err: %v", cron.Namespace, cron.Name, err)
		return nil, err
	}
	return nextScheduledTimeDuration(sched, now), nil
}

func (cc *CronController) refreshCronHistory(cron *v1alpha1.Cron, workloads []metav1.Object) error {
	// history maps current cron history from a unique key to index in history slice.
	history := make(map[string]int)
	// unique cron history keyed by {name}-{startTimestamp}
	key := func(h *v1alpha1.CronHistory) string { return fmt.Sprintf("%s-%d", h.Object.Name, h.Created.Unix()) }

	for i := range cron.Status.History {
		history[key(&cron.Status.History[i])] = i
	}

	for i := range workloads {
		workload := workloads[i]
		newHistory := workloadToHistory(workload, cron.Spec.CronTemplate.APIVersion, cron.Spec.CronTemplate.Kind)
		nk := key(&newHistory)
		if idx, ok := history[nk]; ok {
			// only value of status that can change.
			cron.Status.History[idx].Status = newHistory.Status
			if newHistory.Finished != nil {
				cron.Status.History[idx].Finished = newHistory.Finished
			}
		} else {
			cron.Status.History = append(cron.Status.History, newHistory)
		}
	}

	sort.Sort(byHistoryStartTime(cron.Status.History))

	historySize := len(cron.Status.History)
	if cron.Spec.HistoryLimit != nil && historySize > int(*cron.Spec.HistoryLimit) {
		klog.V(3).Infof("history size of cron %s/%s exceeds history limit %d, truncate it",
			cron.Namespace, cron.Name, *cron.Spec.HistoryLimit)
		cron.Status.History = cron.Status.History[historySize-int(*cron.Spec.HistoryLimit):]
	}

	return cc.client.Status().Update(context.Background(), cron)
}

func (cc *CronController) newWorkloadFromTemplate(cron *v1alpha1.Cron, scheduledTime time.Time) (client.Object, error) {
	// In consideration of workload consistency, we uniformly handle workload decode from RAW bytes
	// instead of type assertion.
	if cron.Spec.CronTemplate.Workload == nil ||
		(cron.Spec.CronTemplate.Workload.Object == nil && cron.Spec.CronTemplate.Workload.Raw == nil) {
		return nil, fmt.Errorf("invalid cron template, neither raw workload bytes nor workload oject were specified, cron: %s", cron.Name)
	}
	wl, err := cc.newEmptyWorkload(cron.Spec.CronTemplate.APIVersion, cron.Spec.CronTemplate.Kind)
	if err != nil {
		return nil, err
	}
	raw := cron.Spec.CronTemplate.Workload.DeepCopy()
	if raw.Raw == nil && raw.Object != nil {
		raw, err = cc.extCodec.EncodeRaw(raw.Object)
		if err != nil {
			return nil, err
		}
	}
	if err = cc.extCodec.DecodeRaw(*raw, wl); err != nil {
		return nil, err
	}
	// modify created workload name when no custom name was predefined.
	if metaWorkload, ok := wl.(metav1.Object); ok {
		if len(metaWorkload.GetGenerateName()) != 0 {
			// Cron does not allow users to set customized generateName, because generated name
			// is suffixed with a randomized string which is not unique, so duplicated scheduling
			// is possible when cron-controller fail-over or fail to update status when workload
			// created, so we forcibly emptied it here.
			metaWorkload.SetGenerateName("")
		}
		if len(metaWorkload.GetName()) == 0 {
			metaWorkload.SetName(getDefaultJobName(cron, scheduledTime))
		} else {
			cc.recorder.Event(cron, corev1.EventTypeNormal, "OverridePolicy", "metadata.name has been specifeid in workload template, override cron concurrency policy as Forbidden")
			cron.Spec.ConcurrencyPolicy = v1.ForbidConcurrent
		}
		label := metaWorkload.GetLabels()
		if label == nil {
			label = make(map[string]string)
		}
		label[v1.LabelCronName] = cron.Name
		metaWorkload.SetLabels(label)
		metaWorkload.SetNamespace(cron.Namespace)
		metaWorkload.SetOwnerReferences(append(metaWorkload.GetOwnerReferences(), *metav1.NewControllerRef(cron, schema.GroupVersionKind{
			Group:   v1alpha1.GroupVersion.Group,
			Version: v1alpha1.GroupVersion.Version,
			Kind:    v1alpha1.KindCron,
		})))
	}
	return wl, err
}

func (cc *CronController) trimFinishedWorkloadsFromActiveList(cron *v1alpha1.Cron, workloads []metav1.Object) error {
	knownActive := sets.NewString()
	for i := range workloads {
		workload := workloads[i]
		knownActive.Insert(string(workload.GetUID()))
		found := inActiveList(cron.Status.Active, workload)
		_, finished := IsWorkloadFinished(workload)
		if !found && !finished {
			// Workload is not in active list and has not finish yet, treat it as an unexpected workload.
			// Retrieve latest cron status and double checks first.
			latestCron := v1alpha1.Cron{}
			if err := cc.client.Get(context.Background(), types.NamespacedName{
				Namespace: cron.Namespace,
				Name:      cron.Name,
			}, &latestCron); err != nil {
				return err
			}
			if inActiveList(latestCron.Status.Active, workload) {
				cron = &latestCron
				continue
			}
			cc.recorder.Eventf(cron, corev1.EventTypeWarning, "ExpectedWorkload", "Saw a workload that the controller did not create or forgot: %s", workload.GetName())
			// We found an unfinished workload that has us as the parent, but it is not in our Active list.
			// This could happen if we crashed right after creating the Workload and before updating the status,
			// or if our workloads list is newer than our cron status after a relist, or if someone intentionally created.
		} else if found && finished {
			deleteFromActiveList(cron, workload.GetUID())
			cc.recorder.Eventf(cron, corev1.EventTypeNormal, "SawCompletedWorkload", "Saw completed workload: %s", workload.GetName())
		}
	}

	// Remove any reference from the active list if the corresponding workload does not exist any more.
	// Otherwise, the cron may be stuck in active mode forever even though there is no matching running.
	for _, ac := range cron.Status.Active {
		if knownActive.Has(string(ac.UID)) {
			continue
		}
		wl, err := cc.newEmptyWorkload(ac.APIVersion, ac.Kind)
		if err != nil {
			klog.Errorf("failed to initialize a new workload, apiVersion: %s, kind: %s, err: %v",
				ac.APIVersion, ac.Kind, err)
			continue
		}
		err = cc.apiReader.Get(context.Background(), types.NamespacedName{Namespace: ac.Namespace, Name: ac.Name}, wl)
		switch {
		case errors.IsNotFound(err):
			cc.recorder.Eventf(cron, corev1.EventTypeNormal, "MissingWorkload", "Active workload went missing: %v", ac.Name)
			deleteFromActiveList(cron, ac.UID)
		case err != nil:
			return err
		}
	}
	return nil
}

func (cc *CronController) listActiveWorkloads(cron *v1alpha1.Cron) ([]metav1.Object, error) {
	// TODO(SimonCqk): also list active workloads by label-selector to achieve completeness, if
	// cron controller immediately exit after create a new workload but before update cron status,
	// the newly created workload will be orphaned.

	active := cron.Status.Active
	workloads := make([]metav1.Object, 0, len(active))

	for i := range active {
		wl, err := cc.newEmptyWorkload(active[i].APIVersion, active[i].Kind)
		if err != nil {
			klog.Errorf("unsupported cron workload and failed to init by scheme, kind: %s, err: %v",
				active[i].Kind, err)
			continue
		}

		if err := cc.client.Get(context.Background(), types.NamespacedName{
			Name:      active[i].Name,
			Namespace: active[i].Namespace,
		}, wl); err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("active workload[%s/%s] in cron[%s/%s] has been deleted.",
					active[i].Namespace, active[i].Name, cron.Namespace, cron.Name)
				continue
			}
			return nil, err
		}
		metaWl, ok := wl.(metav1.Object)
		if !ok {
			klog.Warningf("workload [%s/%s] cannot convert to metav1.Object", active[i].Namespace, active[i].Name)
			continue
		}
		workloads = append(workloads, metaWl)
	}

	return workloads, nil
}

func (cc *CronController) newEmptyWorkload(apiVersion, kind string) (client.Object, error) {
	groupVersion := strings.Split(apiVersion, "/")
	if len(groupVersion) != 2 {
		return nil, fmt.Errorf("unexpected length of apiVersion after split, apiVersion: %s", apiVersion)
	}

	wl, err := cc.scheme.New(schema.GroupVersionKind{
		Group:   groupVersion[0],
		Version: groupVersion[1],
		Kind:    kind,
	})
	if err != nil {
		return nil, err
	}
	if wlo, ok := wl.(client.Object); ok {
		return wlo, nil
	}
	return nil, fmt.Errorf("workload %+v has not implemented client.Object interface", groupVersion)
}

func (cc *CronController) deleteWorkload(cron *v1alpha1.Cron, ref corev1.ObjectReference) error {
	wl, err := cc.newEmptyWorkload(ref.APIVersion, ref.Kind)
	if err != nil {
		return err
	}
	if err = cc.client.Get(context.Background(), types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, wl); err != nil {
		if errors.IsNotFound(err) {
			deleteFromActiveList(cron, ref.UID)
			klog.Infof("workload %s/%s has been deleted", ref.Namespace, ref.Name)
			return nil
		}
		return err
	}
	deleteFromActiveList(cron, ref.UID)
	cc.recorder.Eventf(cron, corev1.EventTypeNormal, "SuccessfulDelete", "successfully delete workload %s", ref.Name)
	return nil
}

var (
	// workloads is a list of cron-able workloads.
	workloads = []client.Object{
		&trainingv1alpha1.TFJob{},
		&trainingv1alpha1.PyTorchJob{},
		&trainingv1alpha1.MarsJob{},
		&trainingv1alpha1.XGBoostJob{},
		&trainingv1alpha1.XDLJob{},
		&trainingv1alpha1.MPIJob{},
		&inferencev1alpha1.ElasticBatchJob{},
	}
)
