#!/bin/bash

topdir="$(cd "$(dirname "$0")/.." || exit; pwd)"
project="github.com/alibaba/kubedl"
apis=$project/api/pytorch/v1,$project/api/tensorflow/v1,$project/api/xdl/v1alpha1,$project/api/xgboost/v1alpha1
output=$project/client
clientset_name="versioned"

go run k8s.io/code-generator/cmd/client-gen \
  --clientset-name "$clientset_name" \
  --input-base "" \
  --input "$apis" \
  --output-package "$output/clientset" \
  --go-header-file "$topdir/hack/boilerplate.go.txt" \
  "$@"

go run k8s.io/code-generator/cmd/lister-gen \
  --input-dirs "$apis" \
  --output-package "$output/listers" \
  --go-header-file "$topdir/hack/boilerplate.go.txt" \
  "$@"

go run k8s.io/code-generator/cmd/informer-gen \
  --input-dirs "$apis" \
  --versioned-clientset-package "$output/clientset/versioned" \
  --listers-package "$output/listers" \
  --output-package "$output/informers" \
  --go-header-file "$topdir/hack/boilerplate.go.txt" \
  "$@"