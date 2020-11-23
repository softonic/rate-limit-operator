IMG ?= softonic/rate-limit-operator:1.0.0
CRD_OPTIONS ?= "crd:trivialVersions=true"
BIN := rate-limit-operator
PKG := github.com/softonic/rate-limit-operator
VERSION ?= 0.1.1
ARCH ?= amd64
APP ?= rate-limit-operator
NAMESPACE ?= rate-limit-operator-system
RELEASE_NAME ?= rate-limit-operator
REPOSITORY ?= softonic/rate-limit-operator

IMAGE := $(BIN)

BUILD_IMAGE ?= golang:1.14-buster

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

.PHONY: all
all: dev

.PHONY: build
build: generate
	go mod download
	GOARCH=${ARCH} go build -ldflags "-X ${PKG}/pkg/version.Version=${VERSION}" .

.PHONY: image
image:
	docker build -t $(IMG) -f Dockerfile .
	docker tag $(IMG) $(REPOSITORY):latest

.PHONY: docker-push
docker-push:
	docker push $(IMG)
	docker push $(REPOSITORY):latest

.PHONY: make-manifest
make-manifest: controller-gen manifests
	docker run --rm -v $(PWD):/app -w /app/ alpine/helm:3.2.3 template --release-name $(RELEASE_NAME) --set "image.tag=$(VERSION)" --set "image.repository=$(REPOSITORY)"  -f chart/rate-limit-operator/values.yaml chart/rate-limit-operator > manifest.yaml

.PHONY: undeploy
undeploy:
	kubectl delete -f manifest.yaml || true

.PHONY: deploy
deploy: make-manifest
	kubectl apply -f manifest.yaml

.PHONY: helm-deploy
helm-deploy: install-crd
	helm upgrade --install $(RELEASE_NAME) --namespace $(NAMESPACE) --set "image.tag=$(VERSION)" -f chart/rate-limit-operator/values.yaml  chart/rate-limit-operator

.PHONY: install-crd
install-crd:
	cp config/crd/bases/networking.softonic.io_ratelimits.yaml chart/rate-limit-operator/crds/networking.softonic.io_ratelimits.yaml

# Run tests
.PHONY: test
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
.PHONY: manager
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
.PHONY: install
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
.PHONY: uninstall
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
.PHONY: deploy
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...

# Generate code
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or download controller-gen
# download controller-gen if necessary
.PHONY: controller-gen
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
