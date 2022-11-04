package core

import (
	"testing"
)

func TestRoundRobinSelector(t *testing.T) {
	testCases := []struct {
		queues        map[string]*queue
		lastIndex     int
		currentQueues []string
		expectedQueue string
	}{
		{
			queues: map[string]*queue{
				"1": {name: "1", created: 1},
			},
			lastIndex:     0,
			expectedQueue: "1",
		},
		{
			queues: map[string]*queue{
				"1": {name: "1", created: 1},
				"2": {name: "2", created: 2},
				"3": {name: "3", created: 3},
			},
			lastIndex:     0,
			currentQueues: []string{"1", "2"},
			expectedQueue: "2",
		},
		{
			queues: map[string]*queue{
				"1": {name: "1", created: 1},
				"2": {name: "2", created: 2},
				"3": {name: "3", created: 3},
			},
			lastIndex:     2,
			currentQueues: []string{"1", "2", "3"},
			expectedQueue: "1",
		},
		{
			queues: map[string]*queue{
				"1": {name: "1", created: 1},
				"2": {name: "2", created: 30},
				"3": {name: "3", created: 15},
				"4": {name: "4", created: 10},
			},
			lastIndex:     2,
			currentQueues: []string{"1", "3", "4"},
			expectedQueue: "2",
		},
	}

	for _, testCase := range testCases {
		rr := roundRobinSelector{
			queueNames:        testCase.currentQueues,
			lastSelectedIndex: testCase.lastIndex,
		}
		nextName, _ := rr.Next(testCase.queues)
		if nextName != testCase.expectedQueue {
			t.Errorf("unexpected queue order, expected: %v, got: %v", testCase.expectedQueue, nextName)
		}
	}
}
