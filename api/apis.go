/*
Copyright 2019 The Alibaba Authors.

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

// Generate deepcopy for xdl apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i ./xdl/... -h ../hack/boilerplate.go.txt
// Generate default for xdl apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/defaulter-gen/main.go -O zz_generated.defaults -i ./xdl/... -h ../hack/boilerplate.go.txt

// Generate deepcopy for tensorflow apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i ./tensorflow/... -h ../hack/boilerplate.go.txt
// Generate default for tensorflow apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/defaulter-gen/main.go -O zz_generated.defaults -i ./tensorflow/... -h ../hack/boilerplate.go.txt

// Generate deepcopy for pytorch apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i ./pytorch/... -h ../hack/boilerplate.go.txt
// Generate default for tensorflow apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/defaulter-gen/main.go -O zz_generated.defaults -i ./pytorch/... -h ../hack/boilerplate.go.txt

// Generate deepcopy for xgboost apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i ./xgboost/... -h ../hack/boilerplate.go.txt
// Generate default for xgboost apis.
//go:generate go run $GOPATH/src/k8s.io/code-generator/cmd/defaulter-gen/main.go -O zz_generated.defaults -i ./xgboost/... -h ../hack/boilerplate.go.txt

package api

import (
	"k8s.io/apimachinery/pkg/runtime"
)

var AddToSchemes runtime.SchemeBuilder

func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
