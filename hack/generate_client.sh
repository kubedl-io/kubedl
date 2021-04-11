#!/usr/bin/env bash

go mod vendor
retVal=$?
if [ $retVal -ne 0 ]; then
    exit $retVal
fi

set -e

GO111MODULE=off /bin/bash vendor/k8s.io/code-generator/generate-groups.sh all \
github.com/alibaba/kubedl/client github.com/alibaba/kubedl/apis "training:v1alpha1" -h ./hack/boilerplate.go.txt

GO111MODULE=off /bin/bash vendor/k8s.io/code-generator/generate-internal-groups.sh defaulter \
github.com/alibaba/kubedl/client github.com/alibaba/kubedl/apis github.com/alibaba/kubedl/apis "training:v1alpha1" -h ./hack/boilerplate.go.txt

rm -rf ./vendor