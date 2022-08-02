package core

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/kubedl/pkg/jobcoordinator"
)

func TestQueueBasicOps(t *testing.T) {
	q := newQueue("test")
	qu1 := newQueueUnitForTest("test-1", "1")
	q.add(qu1)
	assert.Equal(t, q.size(), 1)
	assert.True(t, q.exists("1"))
	qu2 := newQueueUnitForTest("test-2", "2")
	q.add(qu2)
	assert.Equal(t, q.size(), 2)
	assert.True(t, q.exists("2"))
	q.remove("2")
	assert.Equal(t, q.size(), 1)

	tmp := q.get("1")
	assert.True(t, reflect.DeepEqual(tmp, qu1))

	qu1.Priority = pointer.Int32Ptr(100)
	q.update(qu1)
	qu1Copy := q.get("1")
	assert.NotNil(t, qu1Copy)
	assert.NotNil(t, qu1Copy.Priority)
	assert.Equal(t, *qu1Copy.Priority, int32(100))
}

func TestQueueIterator(t *testing.T) {
	q := newQueue("test")
	qu1 := newQueueUnitForTest("test-1", "1")
	qu2 := newQueueUnitForTest("test-2", "2")
	qu3 := newQueueUnitForTest("test-3", "3")

	q.add(qu1)
	q.add(qu2)
	q.add(qu3)

	iter := q.iter()
	cnt := 0
	for iter.HasNext() {
		if cnt == 0 {
			// we remove one queue unit from queue when iterating
			// iterator since iterator holds a queue snapshot and
			// is immutable.
			q.remove("1")
		}
		cnt++
		iter.Next()
	}
	assert.Equal(t, cnt, 3)
}

func newQueueUnitForTest(name string, uid string) *jobcoordinator.QueueUnit {
	return &jobcoordinator.QueueUnit{
		Object: &v1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(uid),
		}},
	}
}
