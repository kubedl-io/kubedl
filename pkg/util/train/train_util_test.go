// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package train

import "testing"

func TestIsRetryableExitCode(t *testing.T) {
	tcs := []struct {
		ExitCode int32
		Expected bool
	}{
		{
			ExitCode: 1,
			Expected: false,
		},
		{
			ExitCode: 2,
			Expected: false,
		},
		{
			ExitCode: 3,
			Expected: false,
		},
		{
			ExitCode: 130,
			Expected: true,
		},
		{
			ExitCode: 138,
			Expected: true,
		},
	}

	for _, tc := range tcs {
		actual := IsRetryableExitCode(tc.ExitCode)
		if actual != tc.Expected {
			t.Errorf("ExitCode %d: Expected %t, got %t", tc.ExitCode, tc.Expected, actual)
		}
	}
}

func TestIsRetryableReason(t *testing.T) {
	tcs := []struct {
		reason   string
		expected bool
	}{
		{
			reason:   "Error",
			expected: false,
		},
		{
			reason:   "OOMKilled",
			expected: true,
		},
		{
			reason:   "Killed",
			expected: true,
		},
		{
			reason:   "Evicted",
			expected: true,
		},
		{
			reason:   "Unknown",
			expected: false,
		},
	}

	for _, tc := range tcs {
		actual := IsRetryablePodFailedReason(tc.reason)
		if actual != tc.expected {
			t.Errorf("Reason %s: Expected %t, got %t", tc.reason, tc.expected, actual)
		}
	}
}
