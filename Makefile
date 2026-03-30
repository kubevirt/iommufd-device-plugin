CONTAINER_ENGINE ?= $(shell KUBEVIRT_CRI=$${KUBEVIRT_CRI} hack/container-engine.sh)
DOCKER_PREFIX ?= quay.io/kubevirt
IMAGE_NAME ?= iommufd-device-plugin
DOCKER_TAG ?= latest
IMG ?= $(DOCKER_PREFIX)/$(IMAGE_NAME):$(DOCKER_TAG)

# Version of golangci-lint to install
GOLANGCI_LINT_VERSION ?= v2.10.1

# Location to install local binaries to
LOCALBIN ?= $(PWD)/_bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
export PATH := $(LOCALBIN):$(PATH)

.PHONY: build
build:
	CGO_ENABLED=0 go build -o iommufd-device-plugin ./cmd/main.go

.PHONY: test
test:
	go test ./pkg/... -v -count=1

.PHONY: lint
lint: golangci-lint
	go vet ./...
	golangci-lint run ./...

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)

.PHONY: image-build
image-build:
	$(CONTAINER_ENGINE) build -t $(IMG) .

.PHONY: image-push
image-push:
	$(CONTAINER_ENGINE) push $(IMG)

.PHONY: image-build-multiarch
image-build-multiarch:
	hack/build-multiarch.sh

.PHONY: image-push-multiarch
image-push-multiarch: image-build-multiarch
	hack/push-multiarch.sh

.PHONY: image-manifest
image-manifest: image-push-multiarch
	hack/push-container-manifest.sh

.PHONY: cluster-up
cluster-up:
	scripts/kubevirtci.sh up

.PHONY: cluster-down
cluster-down:
	scripts/kubevirtci.sh down

.PHONY: cluster-sync
cluster-sync:
	scripts/kubevirtci.sh sync

.PHONY: functest
functest:
	KUBECONFIG=$$(scripts/kubevirtci.sh kubeconfig) go test ./tests/... -v -count=1 -timeout 20m $(FUNC_TEST_ARGS)

.PHONY: clean
clean:
	rm -rf iommufd-device-plugin _bin _kubevirt
