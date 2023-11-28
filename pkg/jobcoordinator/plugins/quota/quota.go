package quota

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	quotautils "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/jobcoordinator"
	"github.com/alibaba/kubedl/pkg/util"
	utilrecorder "github.com/alibaba/kubedl/pkg/util/recorder"
	"github.com/alibaba/kubedl/pkg/util/resource_utils"
	"github.com/alibaba/kubedl/pkg/util/runtime"
)

const (
	Name              = "Quota"
	DefaultTenantName = "default"

	// defaultQuotaAssumedTimeoutSeconds represents the default timeout seconds from
	// resources assumed as requested quota to it becomes invalid/expired.
	defaultQuotaAssumedTimeoutSeconds = float64(60)
)

var quotaPlugin *quota

func New(c client.Client, recorder record.EventRecorder) jobcoordinator.Plugin {
	if quotaPlugin != nil {
		return quotaPlugin
	}
	quotaPlugin = &quota{client: c, recorder: utilrecorder.NewFlowControlledRecorder(recorder, 3)}
	quotaPlugin.assumedQuotas = assumedQuotas{
		assumed:          make(map[string]map[string]assumedQuota),
		latestStatusFunc: quotaPlugin.retrieveLatestStatus,
	}

	return quotaPlugin
}

var _ jobcoordinator.TenantPlugin = &quota{}
var _ jobcoordinator.FilterPlugin = &quota{}
var _ jobcoordinator.PreDequeuePlugin = &quota{}

type quota struct {
	client   client.Client
	recorder record.EventRecorder
	// assumedQuotas represents assumed deducted quota when job is allowed to
	// be dequeued, there always has a latency from job dequeued to requested
	// quota updated.
	assumedQuotas assumedQuotas
}

// TenantName implements tenant interface,
func (q *quota) TenantName(qu *jobcoordinator.QueueUnit) string {
	if qu.SchedPolicy != nil && qu.SchedPolicy.Queue != "" && qu.Tenant == "" {
		qu.Tenant = qu.SchedPolicy.Queue
	}
	if qu.Tenant != "" {
		return qu.Tenant
	}
	// Take namespace as default tenant name since ResourceQuotas is partitioned
	// by namespaces.
	return qu.Object.GetNamespace()
}

func (q *quota) Filter(ctx context.Context, qu *jobcoordinator.QueueUnit) jobcoordinator.Status {

	klog.V(3).Infof("filter queue unit: %s, resources: %+v", qu.Key(), qu.Resources)
	rqs := corev1.ResourceQuotaList{}
	if err := q.client.List(ctx, &rqs, client.InNamespace(qu.Object.GetNamespace())); err != nil {
		klog.Errorf("failed to get quota %s, err: %v", qu.Tenant, err)
		return jobcoordinator.NewStatus(jobcoordinator.Error, err.Error())
	}

	for _, rq := range rqs.Items {
		available, exceed := availableQuota(&rq)
		if exceed {
			klog.V(2).Infof("quota %s has exceeded guaranteed resources", rq.Name)
			return jobcoordinator.NewStatus(jobcoordinator.Wait,
				fmt.Sprintf("queue %s guaranteed quota has exceed", rq.Name))
		}

		assumed := q.assumedQuotas.accumulateAssumedQuota(qu.Tenant)
		if assumed != nil {
			available = quotautils.SubtractWithNonNegativeResult(available, assumed)
		}

		if le, resources := resource_utils.AnyLessThanOrEqual(available, qu.Resources); le {
			msg := fmt.Sprintf("resource %v request exceeds available quota %s, avaiable: %+v, requests: %+v, assumed: %+v",
				resources, qu.Tenant, util.JsonDump(available), util.JsonDump(qu.Resources), util.JsonDump(assumed))
			klog.V(2).Infof(msg)
			q.recorder.Event(qu.Object, corev1.EventTypeNormal, "QuotaNotSatisfied", msg)
			return jobcoordinator.NewStatus(jobcoordinator.Wait, msg)
		}

	}

	return jobcoordinator.NewStatus(jobcoordinator.Success)
}

func (q *quota) PreDequeue(ctx context.Context, qu *jobcoordinator.QueueUnit) jobcoordinator.Status {
	// Assume quota consumed before queue unit was dequeued, because used quota will not immediately
	// be updated util pods created and quota-controller calculates total used then apply to cluster.
	q.assumedQuotas.assume(qu.Tenant, qu)
	return jobcoordinator.NewStatus(jobcoordinator.Success)
}

func (q *quota) Name() string {
	return Name
}

func (q *quota) retrieveLatestStatus(qu *jobcoordinator.QueueUnit) (v1.JobStatus, error) {
	copied := qu.Object.DeepCopyObject().(client.Object)

	if err := q.client.Get(context.Background(), types.NamespacedName{
		Name:      qu.Object.GetName(),
		Namespace: qu.Object.GetNamespace(),
	}, copied); err != nil {
		return v1.JobStatus{}, err
	}

	return runtime.StatusTraitor(copied)
}

func availableQuota(q *corev1.ResourceQuota) (corev1.ResourceList, bool) {
	max := q.Spec.Hard
	requested := q.Status.Used
	requested = quotautils.Mask(requested,
		quotautils.Intersection(quotautils.ResourceNames(requested), quotautils.ResourceNames(max)))
	exceed, _ := resource_utils.AnyLessThanOrEqual(max, requested)
	return quotautils.Subtract(max, requested), exceed
}

type assumedQuotas struct {
	mut sync.RWMutex
	// assumed is a map of assumed quantities of requested quota in case that
	// calculation delay leads to dequeue objects more than expected.
	assumed map[string]map[string]assumedQuota
	// latestStatusFunc retrieves latest queue unit status from cluster.
	latestStatusFunc func(qu *jobcoordinator.QueueUnit) (v1.JobStatus, error)
}

// assume specifies resource quantity has been charged for this
// quota group and will be invalid after defaultQuotaAssumedTimeoutSeconds.
func (aqs *assumedQuotas) assume(quotaName string, qu *jobcoordinator.QueueUnit) {
	aqs.mut.Lock()
	defer aqs.mut.Unlock()

	if aqs.assumed[quotaName] == nil {
		aqs.assumed[quotaName] = make(map[string]assumedQuota)
	}

	klog.V(5).Infof("assume resource %+v of quota %s", qu.Resources, quotaName)
	aqs.assumed[quotaName][qu.Key()] = assumedQuota{
		unit:      qu,
		timestamp: time.Now(),
	}
}

func (aqs *assumedQuotas) cancelWithoutLock(quotaName, key string) {
	delete(aqs.assumed[quotaName], key)
}

// accumulateAssumedQuota aggregates resource quantities assumed for specified
// quota group, it will invalidate stale assumed quotas first before aggregating.
func (aqs *assumedQuotas) accumulateAssumedQuota(name string) corev1.ResourceList {
	aqs.mut.Lock()
	aqs.invalidateAssumedQuotas(name)
	aqs.mut.Unlock()

	aqs.mut.RLock()
	defer aqs.mut.RUnlock()
	var (
		result  corev1.ResourceList
		assumed = aqs.assumed[name]
	)
	for i := range assumed {
		result = quotautils.Add(result, assumed[i].unit.Resources)
	}
	return result
}

// invalidateAssumedQuotas invalidates and release expired assumed quotas.
func (aqs *assumedQuotas) invalidateAssumedQuotas(quotaName string) {
	for key, aq := range aqs.assumed[quotaName] {
		if aq.expired(aqs.latestStatusFunc) {
			klog.V(3).Infof("assumed queue unit %s expired, release it", aq.unit.Key())
			aqs.cancelWithoutLock(quotaName, key)
		}
	}
}

type assumedQuota struct {
	unit      *jobcoordinator.QueueUnit
	timestamp time.Time
}

func (aq *assumedQuota) expired(statusFunc func(qu *jobcoordinator.QueueUnit) (v1.JobStatus, error)) bool {
	// assume timeout.
	if time.Since(aq.timestamp).Seconds() > defaultQuotaAssumedTimeoutSeconds {
		klog.V(5).Infof("queue unit %s resources assumed time exceeds threshold[%v], mark it as expired",
			aq.unit.Key(), defaultQuotaAssumedTimeoutSeconds)
		return true
	}

	// related job has been in running state or finished.
	status, err := statusFunc(aq.unit)
	if err != nil {
		if errors.IsNotFound(err) {
			return true
		}
		klog.Errorf("failed to get latest status for unit %s, err: %v", aq.unit.Key(), err)
	}

	klog.V(4).Infof("latest status for queue unit %s: %v", aq.unit.Key(), util.JsonDump(status))
	if util.IsRunning(status) || util.IsFailed(status) || util.IsSucceeded(status) {
		return true
	}

	return false
}
