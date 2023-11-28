package recorder

import (
	"fmt"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	v1 "github.com/alibaba/kubedl/pkg/test_util/v1"
)

func TestFlowControlledRecorder(t *testing.T) {
	fakeRecorder := record.NewFakeRecorder(10)
	testJob := v1.NewTestJob(1)
	recorder := &flowControlledRecorder{
		qps:            3,
		recorder:       fakeRecorder,
		inflightEvents: make(map[types.UID]inflight),
		notify:         *sync.NewCond(&sync.Mutex{}),
	}
	// 1. test basic emit event
	recorder.Eventf(testJob, corev1.EventTypeNormal, "Hello", "World")
	time.Sleep(10 * time.Millisecond)

	// 2. test emit qps limitation
	start := time.Now()
	for i := 0; i < 7; i++ {
		testJob = v1.NewTestJob(1)
		testJob.UID = types.UID(fmt.Sprintf("%d", i+1))
		recorder.Eventf(testJob, corev1.EventTypeNormal, "Hello", "World")
	}

	// check all events has been consumed.
	for {
		if len(recorder.inflightEvents) == 0 {
			break
		}
	}
	elapse := time.Since(start).Seconds()

	if elapse < 2 {
		t.Errorf("recorder qps limit is 3, emit 7 events but elapse[%v] less than 2 seconds", elapse)
	}
}
