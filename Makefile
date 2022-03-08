NS := github.com/projecteru2/yavirt
BUILD := go build -race
TEST := go test -count=1 -race -cover

LDFLAGS += -X "$(NS)/internal/ver.Git=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(NS)/internal/ver.Compile=$(shell go version)"
LDFLAGS += -X "$(NS)/internal/ver.Date=$(shell date +'%F %T %z')"

PKGS := $$(go list ./... | grep -v -P '$(NS)/third_party|vendor/')

.PHONY: all test build guestfs

default: build

build: build-srv build-ctl

build-srv:
	$(BUILD) -ldflags '$(LDFLAGS)' -o bin/yavirtd yavirtd.go

build-ctl:
	$(BUILD) -ldflags '$(LDFLAGS)' -o bin/yavirtctl cmd/cmd.go

lint: format
	golangci-lint run --skip-dirs-use-default --skip-dirs=third_party

format: vet
	gofmt -s -w $$(find . -iname '*.go' | grep -v -P '\./third_party|\./vendor/')

vet:
	go vet $(PKGS)

deps:
	GO111MODULE=on go mod download
	GO111MODULE=on go mod vendor

mock: deps
	mockery --dir api/image --output api/image/mocks --name PushPuller
	mockery --dir libvirt --output libvirt/mocks --all
	mockery --dir sh --output sh/mocks --name Shell
	mockery --dir store --output store/mocks --name Store
	mockery --dir util --output util/mocks --name Locker
	mockery --dir virt/agent --output virt/agent/mocks --all
	mockery --dir virt/domain --output virt/domain/mocks --name Domain
	mockery --dir virt/guest/manager --output virt/guest/manager/mocks --name Manageable
	mockery --dir virt/guest --output virt/guest/mocks --name Bot
	mockery --dir virt/guestfs --output virt/guestfs/mocks --name Guestfs
	mockery --dir virt/volume --output virt/volume/mocks --name Bot

clean:
	rm -fr bin/*

test:
ifdef RUN
	$(TEST) -v -run='${RUN}' $(PKGS)
else
	$(TEST) $(PKGS)
endif
