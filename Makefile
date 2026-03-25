BINARY := iommufd-device-plugin
IMAGE ?= quay.io/vladikr/iommufd-device-plugin
TAG ?= latest

.PHONY: build image push test clean

build:
	CGO_ENABLED=0 go build -o $(BINARY) ./cmd/main.go

image:
	podman build -t $(IMAGE):$(TAG) .

push:
	podman push $(IMAGE):$(TAG)

test:
	go test -v ./...

clean:
	rm -f $(BINARY)
