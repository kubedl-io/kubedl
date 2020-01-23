package workloadgate

import (
	"reflect"
	"testing"
)

func TestIsWorkloadEnable(t *testing.T) {
	cases := []struct {
		workloads       string
		expectEnables   map[string]bool
		expectEnableAll bool
	}{
		{
			workloads:       "",
			expectEnables:   map[string]bool{},
			expectEnableAll: false,
		},
		{
			workloads:       "*",
			expectEnables:   map[string]bool{},
			expectEnableAll: true,
		},
		{
			workloads:       "*,foo",
			expectEnables:   map[string]bool{"foo": true},
			expectEnableAll: true,
		},
		{
			workloads:       "foo,*",
			expectEnables:   map[string]bool{"foo": true},
			expectEnableAll: true,
		},
		{
			workloads:       "foo,a",
			expectEnables:   map[string]bool{"foo": true, "a": true},
			expectEnableAll: false,
		},
		{
			workloads:       "foo,-a",
			expectEnables:   map[string]bool{"foo": true, "a": false},
			expectEnableAll: false,
		},
		{
			workloads:       "-foo,a",
			expectEnables:   map[string]bool{"foo": false, "a": true},
			expectEnableAll: false,
		},
		{
			workloads:       "foo,-*",
			expectEnables:   map[string]bool{"foo": true},
			expectEnableAll: false,
		},
	}
	for _, c := range cases {
		enables, enableAll := parseWorkloadsEnabled(c.workloads)
		if !reflect.DeepEqual(enables, c.expectEnables) {
			t.Fatalf("workloads: %s, expected: %v, got: %v", c.workloads, c.expectEnables, enables)
		}
		if enableAll != c.expectEnableAll {
			t.Fatalf("workloads %s, expected: %v, got: %v", c.workloads, c.expectEnableAll, enableAll)
		}
	}
}
