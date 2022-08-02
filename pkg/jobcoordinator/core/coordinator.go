package core

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sync"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/alibaba/kubedl/pkg/jobcoordinator"
	"github.com/alibaba/kubedl/pkg/jobcoordinator/helper"
	coordinatorplugins "github.com/alibaba/kubedl/pkg/jobcoordinator/plugins"
	"github.com/alibaba/kubedl/pkg/util"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type Coordinator interface {
	Run(ctx context.Context)
	EnqueueOrUpdate(qu *jobcoordinator.QueueUnit)
	Dequeue(uid types.UID)
	IsQueuing(uid types.UID) bool
	SetQueueUnitOwner(uid types.UID, owner workqueue.RateLimitingInterface)
}

var globalCoordinator Coordinator

func NewCoordinator(mgr ctrl.Manager) Coordinator {
	if globalCoordinator != nil {
		return globalCoordinator
	}

	clientset := mgr.GetClient()
	recorder := mgr.GetEventRecorderFor("JobCoordinator")
	co := &coordinator{
		config:           coordinatorplugins.NewCoordinatorConfiguration(),
		queues:           make(map[string]*queue),
		queueStateMarker: helper.QueueStateMarker(clientset, recorder),
		selector:         NewRoundRobinSelector(),
		queueIndexer:     newQueueIndexer(),
	}

	var err error
	pluginRegistry := coordinatorplugins.NewPluginsRegistry()
	setDefaultCoordinatorPlugins(co, clientset, recorder)

	if tp, err := newTenantPlugin(clientset, recorder, pluginRegistry, co.config.TenantPlugin); err != nil {
		log.Error(err, "failed to new tenant plugin")
	} else {
		co.tenantPlugin = tp
	}

	if err = updatePluginList(clientset, recorder, &co.prefilterPlugins, pluginRegistry, co.config.PreFilterPlugins); err != nil {
		log.Error(err, "failed to new prefilter plugins")
	}
	if err = updatePluginList(clientset, recorder, &co.filterPlugins, pluginRegistry, co.config.FilterPlugins); err != nil {
		log.Error(err, "failed to new filter plugins")
	}
	if err = updatePluginList(clientset, recorder, &co.scorePlugins, pluginRegistry, co.config.ScorePlugins); err != nil {
		log.Error(err, "failed to new score plugins")
	}
	if err = updatePluginList(clientset, recorder, &co.preDequeuePlugins, pluginRegistry, co.config.PreDequeuePlugins); err != nil {
		log.Error(err, "failed to new pre-dequeue plugins")
	}

	globalCoordinator = co
	go co.Run(context.Background())
	return globalCoordinator
}

var (
	_   Coordinator = &coordinator{}
	log             = logf.Log.WithName("job-coordinator")
)

type coordinator struct {
	config jobcoordinator.CoordinatorConfiguration
	// lock protects fields down below when concurrent reads/writes happen.
	lock sync.RWMutex
	// queues represents a slice of tenant sub-queues who holds kinds of jobs belongs to one tenant
	// and waiting to be scheduled.
	// tenant is partitioned by TenantPlugin implementation, for example, queue can be partitioned
	// by quota, and each quota represents an independent tenant.
	queues map[string]*queue
	// queueIndexer knows how to find target queue index in slice by object uid or queue name.
	queueIndexer *queueIndexer
	// selector knows how to select next queue.
	selector QueueSelector
	// queueStateMarker patches job-in-queue as status queueing with enqueued/dequeued reason.
	queueStateMarker func(qu *jobcoordinator.QueueUnit, reason string) error

	// plugins embedded in coordinator and called in extensions points.
	tenantPlugin      jobcoordinator.TenantPlugin
	prefilterPlugins  []jobcoordinator.PreFilterPlugin
	filterPlugins     []jobcoordinator.FilterPlugin
	scorePlugins      []jobcoordinator.ScorePlugin
	preDequeuePlugins []jobcoordinator.PreDequeuePlugin
}

func (co *coordinator) Run(ctx context.Context) {
	wait.UntilWithContext(ctx, co.schedule, co.config.SchedulePeriod)
}

func (co *coordinator) EnqueueOrUpdate(qu *jobcoordinator.QueueUnit) {
	qu.Tenant = co.tenantPlugin.TenantName(qu)

	co.initializeNewQueue(qu)

	if err := co.queueStateMarker(qu, util.JobEnqueuedReason); err != nil {
		log.Error(err, "failed to update queue-unit as queuing and enqueued reason")
	}

	co.addNewQueueUnit(qu.Tenant, qu)

	log.Info("queue unit successfully enqueued", "queue name", qu.Tenant, "key", qu.Key())
}

func (co *coordinator) Dequeue(uid types.UID) {
	qu := co.popQueueUnitFromQueue(uid)
	if qu == nil || qu.Object == nil {
		return
	}

	qu.Owner.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: qu.Object.GetNamespace(),
			Name:      qu.Object.GetName(),
		}})

	if err := co.queueStateMarker(qu, util.JobDequeuedReason); err != nil {
		log.Error(err, "failed to update queue-unit as queuing and dequeued reason")
	}
	log.Info("queue unit successfully dequeued", "queue name", qu.Tenant, "key", qu.Key())
}

func (co *coordinator) SetQueueUnitOwner(uid types.UID, owner workqueue.RateLimitingInterface) {
	co.lock.Lock()
	defer co.lock.Unlock()

	tenant, ok := co.queueIndexer.lookup(uid)
	if !ok {
		log.Info("[SetQueueUnitOwner] can not find queue, ignore it", "uid", uid)
		return
	}

	qu := co.queues[tenant].get(uid)
	qu.Owner = owner
	co.queues[tenant].update(qu)
}

func (co *coordinator) IsQueuing(uid types.UID) bool {
	co.lock.RLock()
	defer co.lock.RUnlock()

	tenant, ok := co.queueIndexer.lookup(uid)
	if !ok {
		return false
	}
	return co.queues[tenant].exists(uid)
}

func (co *coordinator) schedule(ctx context.Context) {
	tenant, q := co.nextQueueToSchedule()
	if q == nil {
		log.V(6).Info("no queue available yet, wait for next request")
		return
	}

	log.V(2).Info("start to schedule next queue", "tenant", tenant, "queue size", q.size())

	iter := co.queueSnapshot(q)

	candidates := make([]jobcoordinator.QueueUnitScore, 0, q.size())

	for iter.HasNext() {
		qu := iter.Next()
		if qu.Owner == nil {
			// Owner queue pointer has not been assigned yet due to schedule triggers before SetQueueUnitOwner
			// invoked, since they are running in parallel goroutines.
			continue
		}

		log.V(3).Info("start to schedule job in queue", "job", qu.Key(), "tenant", tenant)

		if !util.IsEnqueued(*qu.Status) {
			if err := co.queueStateMarker(qu, util.JobEnqueuedReason); err != nil {
				log.Error(err, "failed to update queue-unit as queuing and enqueued reason")
			}
		}

		if co.isQueueUnitAcceptable(ctx, qu) {
			quScore := co.prioritizeQueueUnit(ctx, qu)
			candidates = append(candidates, quScore)
		}
	}

	if len(candidates) == 0 {
		log.V(5).Info("empty feasible queue unit after filtering and scoring, scheduling cycle is being interrupted", "tenant", tenant)
		return
	}

	selected := co.selectQueueUnit(candidates)

	// Run pre dequeue plugins before dequeue queue unit to its owner queue.
	if len(co.preDequeuePlugins) > 0 {
		for _, pg := range co.preDequeuePlugins {
			pg.PreDequeue(ctx, selected.QueueUnit)
		}
	}

	log.V(3).Info("selected queue unit to be dequeued and reconciled", "key", selected.QueueUnit.Key(), "score", selected.Score)
	co.Dequeue(selected.QueueUnit.Object.GetUID())
}

// isQueueUnitAcceptable runs all registered filter plugins and filter out queue units can be
// acceptable by the controller workqueue.
func (co *coordinator) isQueueUnitAcceptable(ctx context.Context, qu *jobcoordinator.QueueUnit) bool {
	if len(co.filterPlugins) == 0 {
		return true
	}

	feasible := true
	for _, fp := range co.prefilterPlugins {
		log.V(3).Info("run filter plugin for queue unit", "plugin name", fp.Name(), "key", qu.Key())

		status := fp.PreFilter(ctx, qu)
		if !status.IsSuccess() {
			if status.Code() == jobcoordinator.Skip {
				log.V(3).Info("skip queue unit", "key", qu.Key(), "reason", status.Reasons())
				continue
			}
			log.V(3).Info("[PreFilter] queue unit is not feasible for current scheduling cycle", "reason", status.Message())
			feasible = false
		}
	}
	if !feasible {
		return false
	}

	for _, fp := range co.filterPlugins {
		log.V(3).Info("run filter plugin for queue unit", "plugin name", fp.Name(), "key", qu.Key())

		status := fp.Filter(ctx, qu)
		if !status.IsSuccess() {
			if status.Code() == jobcoordinator.Skip {
				log.V(3).Info("skip queue unit", "key", qu.Key(), "reason", status.Reasons())
				continue
			}
			log.V(3).Info("[Filter]queue unit is not feasible for current scheduling cycle", "reason", status.Message())
			feasible = false
		}
	}

	return feasible
}

// prioritizeQueueUnit runs all registered score plugins and accumulates an integer indicating
// the rank of the queue unit.
func (co *coordinator) prioritizeQueueUnit(ctx context.Context, qu *jobcoordinator.QueueUnit) jobcoordinator.QueueUnitScore {
	qus := jobcoordinator.QueueUnitScore{QueueUnit: qu}

	if len(co.scorePlugins) == 0 {
		return qus
	}

	for _, sp := range co.scorePlugins {
		log.V(3).Info("run score plugin for queue unit", "plugin name", sp.Name(), "key", qu.Key())

		score, status := sp.Score(ctx, qu)
		if !status.IsSuccess() {
			log.V(3).Info("queue unit does not fit current scheduling cycle", "reason", status.Message())
			continue
		}
		qus.Score += score
	}

	return qus
}

// selectQueueUnit selects a queue unit with highest score to be dequeued, if more than one
// unit gets highest score, it will return a randomly selected one.
func (co *coordinator) selectQueueUnit(candidates []jobcoordinator.QueueUnitScore) jobcoordinator.QueueUnitScore {
	maxScore := candidates[0].Score
	selectedIndex := 0
	cntOfMaxScore := 1
	for i, ns := range candidates[1:] {
		index := i + 1
		if ns.Score > maxScore {
			maxScore = ns.Score
			selectedIndex = index
			cntOfMaxScore = 1
		} else if ns.Score == maxScore {
			cntOfMaxScore++
			if rand.Intn(cntOfMaxScore) == 0 {
				// Replace the candidate with probability of 1/cntOfMaxScore
				selectedIndex = index
			}
		}
	}
	return candidates[selectedIndex]
}

func (co *coordinator) nextQueueToSchedule() (string, *queue) {
	qn, q := co.selector.Next(co.queues)
	if q != nil {
		coordinatorQueuePendingJobsCount.WithLabelValues(qn).Set(float64(q.size()))
	}
	return qn, q
}

func (co *coordinator) initializeNewQueue(qu *jobcoordinator.QueueUnit) {
	co.lock.Lock()
	defer co.lock.Unlock()

	if _, ok := co.queues[qu.Tenant]; !ok {
		co.queues[qu.Tenant] = newQueue(qu.Tenant)
		log.V(2).Info("new queue created", "queue name", qu.Tenant)
	}
	co.queueIndexer.insert(qu.Object.GetUID(), qu.Tenant)
}

func (co *coordinator) addNewQueueUnit(tenant string, qu *jobcoordinator.QueueUnit) {
	co.lock.Lock()
	defer co.lock.Unlock()

	co.queues[tenant].add(qu)
}

func (co *coordinator) popQueueUnitFromQueue(uid types.UID) *jobcoordinator.QueueUnit {
	co.lock.Lock()
	defer co.lock.Unlock()

	tenant, exist := co.queueIndexer.lookup(uid)
	if !exist {
		log.Info("[Dequeue] queue unit has already dequeued, ignore it", "uid", uid)
		return nil
	}

	qu := co.queues[tenant].get(uid)
	co.queues[tenant].remove(uid)
	co.queueIndexer.remove(uid)
	return qu
}

func (co *coordinator) queueSnapshot(q *queue) Iterator {
	co.lock.RLock()
	defer co.lock.RUnlock()
	return q.iter()
}

func updatePluginList(c client.Client, recorder record.EventRecorder, pluginList interface{}, registry coordinatorplugins.Registry, enablePlugins []string) error {
	if len(enablePlugins) == 0 {
		return nil
	}

	plugins := reflect.ValueOf(pluginList).Elem()
	pluginType := plugins.Type().Elem()
	set := sets.NewString()
	for _, ep := range enablePlugins {
		pgFactory, ok := registry[ep]
		if !ok {
			return fmt.Errorf("%s %q does not exist", pluginType.Name(), ep)
		}

		pg := pgFactory(c, recorder)

		if !reflect.TypeOf(pg).Implements(pluginType) {
			return fmt.Errorf("plugin %q does not extend %s plugin", ep, pluginType.Name())
		}

		if set.Has(ep) {
			return fmt.Errorf("plugin %q already registered as %q", ep, pluginType.Name())
		}

		set.Insert(ep)

		newPlugins := reflect.Append(plugins, reflect.ValueOf(pg))
		plugins.Set(newPlugins)
	}
	return nil
}

func newTenantPlugin(c client.Client, recorder record.EventRecorder, registry coordinatorplugins.Registry, enablePlugin string) (jobcoordinator.TenantPlugin, error) {

	pgFactory, ok := registry[enablePlugin]
	if !ok {
		return nil, fmt.Errorf("%s does not exist", enablePlugin)
	}

	pg := pgFactory(c, recorder)
	if !reflect.TypeOf(pg).Implements(reflect.TypeOf((*jobcoordinator.TenantPlugin)(nil)).Elem()) {
		return nil, fmt.Errorf("plugin %q does not extend TenantPlugin plugin", enablePlugin)
	}

	return pg.(jobcoordinator.TenantPlugin), nil
}

func newQueueIndexer() *queueIndexer {
	return &queueIndexer{
		data: make(map[types.UID]string),
	}
}

type queueIndexer struct {
	data map[types.UID]string
}

func (qi *queueIndexer) lookup(uid types.UID) (string, bool) {
	tenant, ok := qi.data[uid]
	return tenant, ok
}
func (qi *queueIndexer) insert(uid types.UID, tenant string) {
	if qi.data == nil {
		qi.data = make(map[types.UID]string)
	}
	qi.data[uid] = tenant
}
func (qi *queueIndexer) remove(uid types.UID) {
	delete(qi.data, uid)
}
