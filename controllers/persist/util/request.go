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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IDName(obj metav1.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetUID(), obj.GetName())
}

func ParseIDName(key string) (id, name string, err error) {
	parsed := strings.Split(key, "/")
	if len(parsed) != 2 {
		return "", "", fmt.Errorf("invalid key: %s, valid format is id/name", err)
	}
	id = strings.TrimSpace(parsed[0])
	name = strings.TrimSpace(parsed[1])
	return id, name, nil
}
