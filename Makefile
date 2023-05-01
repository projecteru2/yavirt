NS := github.com/projecteru2/yavirt
BUILD := go build -race
TEST := go test -count=1 -race -cover

LDFLAGS += -X "$(NS)/internal/ver.Git=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(NS)/internal/ver.Compile=$(shell go version)"
LDFLAGS += -X "$(NS)/internal/ver.Date=$(shell date +'%F %T %z')"

PKGS := $$(go list ./... | grep -v -P '$(NS)/third_party|vendor/')

.PHONY: all test build setup

default: build

build: build-srv build-ctl

build-srv:
	$(BUILD) -ldflags '$(LDFLAGS)' -o bin/yavirtd yavirtd.go

build-ctl:
	$(BUILD) -ldflags '$(LDFLAGS)' -o bin/yavirtctl cmd/cmd.go

setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/vektra/mockery/v2@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

lint: format
	PATH=${HOME}/go/bin:${PATH} golangci-lint run --skip-dirs-use-default --skip-dirs=thirdparty

format: vet
	gofmt -s -w $$(find . -iname '*.go' | grep -v -P '\./third_party|\./vendor/')

vet:
	go vet $(PKGS)

deps:
	go mod tidy

mock: deps
	PATH=${HOME}/go/bin:${PATH} mockery --dir pkg/libvirt --output pkg/libvirt/mocks --all
	PATH=${HOME}/go/bin:${PATH} mockery --dir pkg/sh --output pkg/sh/mocks --name Shell
	PATH=${HOME}/go/bin:${PATH} mockery --dir pkg/store --output pkg/store/mocks --name Store
	PATH=${HOME}/go/bin:${PATH} mockery --dir pkg/utils --output pkg/utils/mocks --name Locker
	PATH=${HOME}/go/bin:${PATH} mockery --dir internal/virt/agent --output internal/virt/agent/mocks --all
	PATH=${HOME}/go/bin:${PATH} mockery --dir internal/virt/domain --output internal/virt/domain/mocks --name Domain
	PATH=${HOME}/go/bin:${PATH} mockery --dir internal/virt/guest/manager --output internal/virt/guest/manager/mocks --name Manageable
	PATH=${HOME}/go/bin:${PATH} mockery --dir internal/virt/guest --output internal/virt/guest/mocks --name Bot
	PATH=${HOME}/go/bin:${PATH} mockery --dir internal/virt/guestfs --output internal/virt/guestfs/mocks --name Guestfs
	PATH=${HOME}/go/bin:${PATH} mockery --dir internal/virt/volume --output internal/virt/volume/mocks --name Bot

clean:
	rm -fr bin/*

test:
ifdef RUN
	$(TEST) -v -run='${RUN}' $(PKGS)
else
	$(TEST) $(PKGS)
endif
