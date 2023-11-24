
# Image URL to use all building/pushing image targets
VERSION ?= daily
HELM_CHART_VERSION ?= 0.1.0
IMG ?= kubedl/kubedl:$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,maxDescLen=0,generateEmbeddedObjectMeta=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet
	KUBEDL_CI=true go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./apis/..." paths="./controllers/..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./apis/..." paths="./controllers/..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

dashboard-image-build:
	docker build . -f Dockerfile.dashboard -t kubedl/dashboard:daily

dashboard-image-push:
	docker push kubedl/dashboard:daily

# Update helm charts
# For example: export VERSION=0.5.0 && make helm-chart
helm-chart:
	cp config/crd/bases/* helm/kubedl/crds
	# generate .kubedbackup file is for compatible between MAC OS and LINUX, since sed syntax is different
	sed -i.kubedlbackup 's/tag:.*/tag: '"$(VERSION)"'/g' helm/kubedl/values.yaml
	sed -i.kubedlbackup 's/version:.*/version: '"$(HELM_CHART_VERSION)"'/g' helm/kubedl/Chart.yaml
	sed -i.kubedlbackup 's/appVersion:.*/appVersion: '"$(VERSION)"'/g' helm/kubedl/Chart.yaml
	cp config/rbac/role.yaml helm/kubedl/templates
	sed -i.kubedlbackup 's/name:.*/name: {{ include "kubedl.fullname" . }}-role/g' helm/kubedl/templates/role.yaml
	rm -f helm/kubedl/*.kubedlbackup
	rm -f helm/kubedl/templates/*.kubedlbackup

# When making a release, e.g. 0.4.0, run below commands:
# 1. create a new git branch: release-0.4.0
# 2. export VERSION=0.4.0 && make release
# 2. push the release-0.4.0 branch
release: helm-chart
	# Set the VERSION across YAML files
	sed -i.kubedlbackup 's/daily/'"$(VERSION)"'/g' config/manager/all_in_one.yaml
	sed -i.kubedlbackup 's/daily/'"$(VERSION)"'/g' config/manager/kustomization.yaml
	rm -f config/manager/*.kubedlbackup

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.1
	go install sigs.k8s.io/controller-tools/cmd/controller-gen
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
