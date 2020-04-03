/*
Copyright 2020 The Alibaba Authors.

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

package util

import (
	"time"
)

// TimePtr returns a pointer to a Time.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// Time returns a Time object from a Time pointer.
func Time(s *time.Time) time.Time {
	if s == nil {
		return time.Time{}
	}
	return *s
}

// IntPtr returns a pointer to integer.
func IntPtr(i int) *int {
	return &i
}

// UInt32Ptr returns a pointer to uint32.
func UInt32Ptr(i uint32) *uint32 {
	return &i
}
