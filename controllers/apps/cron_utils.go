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
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	"github.com/alibaba/kubedl/apis/apps/v1alpha1"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/util"
	utilruntime "github.com/alibaba/kubedl/pkg/util/runtime"
)

var (
	nextScheduleDelta = 100 * time.Millisecond
)

// nextScheduledTimeDuration returns the time duration to requeue based on
// the schedule and current time. It adds a 100ms padding to the next requeue to account
// for Network Time Protocol(NTP) time skews. If the time drifts are adjusted which in most
// realistic cases would be around 100s, scheduled cron will still be executed without missing
// the schedule.
func nextScheduledTimeDuration(sched cron.Schedule, now time.Time) *time.Duration {
	t := sched.Next(now).Add(nextScheduleDelta).Sub(now)
	return &t
}

// getNextScheduleTime gets the time of next schedule after last scheduled and before now it
// returns nil if no unmet schedule times.
// If there are too many (>100) unstarted times, it will raise a warning and but still return
// the list of missed times.
func getNextScheduleTime(cron *v1alpha1.Cron, now time.Time, schedule cron.Schedule, recorder record.EventRecorder) (*time.Time, error) {
	var (
		earliestTime time.Time
	)

	if cron.Status.LastScheduleTime != nil {
		earliestTime = cron.Status.LastScheduleTime.Time
	} else {
		earliestTime = cron.ObjectMeta.CreationTimestamp.Time
	}
	if earliestTime.After(now) {
		return nil, nil
	}

	t, numberOfMissedSchedules, err := getMostRecentScheduleTime(earliestTime, now, schedule)

	if numberOfMissedSchedules > 100 {
		// An object might miss several starts. For example, if
		// controller gets wedged on friday at 5:01pm when everyone has
		// gone home, and someone comes in on tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly cronJob, should
		// all start running with no further intervention (if the cronJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		recorder.Eventf(cron, corev1.EventTypeWarning, "TooManyMissedTimes", "too many missed start times: %d. Check clock skew", numberOfMissedSchedules)
		klog.Infof("too many missed times, cron: %s/%s, missed times: %v", cron.Namespace, cron.Name, numberOfMissedSchedules)
	}
	return t, err
}

// getMostRecentScheduleTime returns the latest schedule time between earliestTime and the count of number of
// schedules in between them
func getMostRecentScheduleTime(earliestTime time.Time, now time.Time, schedule cron.Schedule) (*time.Time, int64, error) {
	t1 := schedule.Next(earliestTime)
	t2 := schedule.Next(t1)

	if now.Before(t1) {
		return nil, 0, nil
	}
	if now.Before(t2) {
		return &t1, 1, nil
	}

	// It is possible for cron.ParseStandard("59 23 31 2 *") to return an invalid schedule
	// seconds - 59, minute - 23, hour - 31 (?!)  dom - 2, and dow is optional, clearly 31 is invalid
	// In this case the scheduleDelta will be 0, and we error out the invalid schedule.
	scheduleDelta := int64(t2.Sub(t1).Round(time.Second).Seconds())
	if scheduleDelta < 1 {
		return nil, 0, fmt.Errorf("time difference between two schedules less than 1 second")
	}

	// At this point, now > t2 > t1, which means cron has missed more than one time.
	for !t2.After(now) {
		t2 = schedule.Next(t2)
	}

	timeElapsed := int64(now.Sub(t1).Seconds())
	numberOfMissedSchedules := (timeElapsed / scheduleDelta) + 1
	t := time.Unix(t1.Unix()+((numberOfMissedSchedules-1)*scheduleDelta), 0).UTC()
	return &t, numberOfMissedSchedules, nil
}

func inActiveList(active []corev1.ObjectReference, workload metav1.Object) bool {
	for i := range active {
		if active[i].UID == workload.GetUID() {
			return true
		}
	}
	return false
}

func workloadToHistory(wl metav1.Object, apiGroup, kind string) v1alpha1.CronHistory {
	status, finished := IsWorkloadFinished(wl)
	created := wl.GetCreationTimestamp()
	ch := v1alpha1.CronHistory{
		Created: &created,
		Status:  status,
		Object: corev1.TypedLocalObjectReference{
			APIGroup: &apiGroup,
			Kind:     kind,
			Name:     wl.GetName(),
		},
	}
	if finished {
		// Finished is the finished status observed timestamp of controller view.
		now := metav1.Now()
		ch.Finished = &now
	}
	return ch
}

func deleteFromActiveList(cron *v1alpha1.Cron, uid types.UID) {
	if cron == nil {
		return
	}
	uidIndex := -1
	for i := 0; i < len(cron.Status.Active); i++ {
		if cron.Status.Active[i].UID == uid {
			uidIndex = i
			break
		}
	}
	if uidIndex < 0 {
		return
	}
	copy(cron.Status.Active[uidIndex:], cron.Status.Active[uidIndex+1:])
	cron.Status.Active = cron.Status.Active[:len(cron.Status.Active)-1]
}

func IsWorkloadFinished(workload metav1.Object) (v1.JobConditionType, bool) {
	status, err := utilruntime.StatusTraitor(workload)
	if err != nil {
		klog.Errorf("failed to trait status from workload, err: %v", err)
		return "", false
	}
	finished := util.IsFailed(status) || util.IsSucceeded(status)
	if len(status.Conditions) > 0 {
		return status.Conditions[len(status.Conditions)-1].Type, finished
	}
	return "", finished
}

func getDefaultJobName(c *v1alpha1.Cron, scheduledTime time.Time) string {
	return fmt.Sprintf("%s-%d", c.Name, scheduledTime.Unix())
}

// cronHistory sorts a list of history by start timestamp, using their names as a tie breaker.
type byHistoryStartTime []v1alpha1.CronHistory

func (o byHistoryStartTime) Len() int      { return len(o) }
func (o byHistoryStartTime) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o byHistoryStartTime) Less(i, j int) bool {
	if o[i].Created == nil && o[j].Created != nil {
		return false
	}
	if o[i].Created != nil && o[j].Created == nil {
		return true
	}
	if o[i].Created.Equal(o[j].Created) {
		return o[i].Object.Name < o[j].Object.Name
	}
	return o[i].Created.Before(o[j].Created)
}
