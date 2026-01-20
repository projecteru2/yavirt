ifeq ($(CN), 1)
ENV := GOPROXY=https://goproxy.cn,direct
endif

NS := github.com/projecteru2/yavirt
BUILD := go build -race
TEST := go test -count=1 -race -cover -gcflags=all=-l

REVISION := $(shell git rev-parse HEAD || unknown)
BUILTAT := $(shell date +%Y-%m-%dT%H:%M:%S)
VERSION := $(shell git describe --tags $(shell git rev-list --tags --max-count=1))

LDFLAGS += -X "$(NS)/internal/ver.REVISION=$(REVISION)"
LDFLAGS += -X "$(NS)/internal/ver.BUILTAT=$(BUILTAT)"
LDFLAGS += -X "$(NS)/internal/ver.VERSION=$(VERSION)"

PKGS := $$(go list ./... | grep -v -P '$(NS)/third_party|vendor/|mocks|ovn')

.PHONY: all test build setup

default: build

build: build-srv build-ctl

build-srv:
	$(BUILD) -ldflags '$(LDFLAGS)' -o bin/yavirtd yavirtd.go

build-ctl:
	$(BUILD) -ldflags '$(LDFLAGS)' -o bin/yavirtctl cmd/cmd.go

setup: setup-lint
	$(ENV) go install github.com/vektra/mockery/v2@latest

setup-lint:
	$(ENV) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.8.0

lint: format
	golangci-lint --version
	golangci-lint run

format: vet
	gofmt -s -w $$(find . -iname '*.go' | grep -v -P '\./third_party|\./vendor/')

vet:
	go vet $(PKGS)

deps:
	$(ENV) go mod tidy
	$(ENV) go mod vendor

mock: deps
	mockery --dir pkg/libvirt --output pkg/libvirt/mocks --all
	mockery --dir pkg/sh --output pkg/sh/mocks --name Shell
	mockery --dir pkg/store --output pkg/store/mocks --name Store
	mockery --dir pkg/utils --output pkg/utils/mocks --name Locker
	mockery --dir pkg/vmimage --output mocks --name Manager
	mockery --dir internal/virt/agent --output internal/virt/agent/mocks --all
	mockery --dir internal/virt/domain --output internal/virt/domain/mocks --name Domain
	mockery --dir internal/virt/guest --output internal/virt/guest/mocks --name Bot
	mockery --dir internal/virt/guestfs --output internal/virt/guestfs/mocks --name Guestfs
	mockery --dir internal/volume --output internal/volume/mocks --name Volume
	mockery --dir internal/volume/base --output internal/volume/base/mocks --name SnapshotAPI
	mockery --dir internal/eru/store --output internal/eru/store/mocks --name Store
	mockery --dir internal/service --output internal/service/mocks --name Service

clean:
	rm -fr bin/*

test:
ifdef RUN
	$(TEST) -v -run='${RUN}' $(PKGS)
else
	$(TEST) $(PKGS)
endif
