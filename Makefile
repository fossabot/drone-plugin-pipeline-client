EXECUTABLE ?= pipeline-client
IMAGE ?= banzaicloud/$(EXECUTABLE)
TAG ?= $(shell git describe --tags --abbrev=0)

LD_FLAGS = -X "main.version=$(TAG)"
PACKAGES = $(shell go list ./... | grep -v /vendor/)

.DEFAULT_GOAL := list

.PHONY: list
list:
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

all: clean deps fmt vet build

clean:
	rm -rf bin/*
	go clean -i ./...

deps:
	go get ./...

fmt:
	go fmt $(PACKAGES)

vet:
	go vet $(PACKAGES)

docker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w $(LD_FLAGS)' -o bin/linux-amd64/$(EXECUTABLE)
	docker build --rm -t $(IMAGE) .
	docker tag $(IMAGE):latest $(IMAGE):$(TAG)

push:
	docker push $(IMAGE):latest
	docker push $(IMAGE):$(TAG)

$(EXECUTABLE): $(wildcard *.go)
	go build -ldflags '-s -w $(LD_FLAGS)' -o bin/$(EXECUTABLE)

build: $(EXECUTABLE)
