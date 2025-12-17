#!/usr/bin/make -f

PACKAGES_SIMTEST=$(shell go list ./... | grep '/simulation')
VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
LEDGER_ENABLED ?= true
SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')
BUILDDIR ?= $(CURDIR)/build
DOCKER := $(shell which docker)
BINDIR ?= $(GOPATH)/bin

# process build tags
build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += gcc cleveldb
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=interbank-netting \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=interbank-nettingd \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq ($(LINK_STATICALLY),true)
	ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static"
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

# The below include contains the tools and runsim targets.
include contrib/devtools/Makefile

###############################################################################
###                                  Build                                  ###
###############################################################################

all: install lint test

build: go.sum
ifeq ($(OS),Windows_NT)
	go build -mod=readonly $(BUILD_FLAGS) -o build/interbank-nettingd.exe ./cmd/interbank-nettingd
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/interbank-nettingd ./cmd/interbank-nettingd
endif

build-reproducible: go.sum
	$(DOCKER) buildx create --name interbank-netting-builder || true
	$(DOCKER) buildx use interbank-netting-builder
	$(DOCKER) buildx build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg GIT_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(COMMIT) \
		--build-arg RUNNER_IMAGE=gcr.io/distroless/static \
		--platform linux/amd64,linux/arm64 \
		-t interbank-netting:$(VERSION) \
		--load \
		-f Dockerfile .

install: go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/interbank-nettingd

###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.11.6
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

###############################################################################
###                                 Tests                                   ###
###############################################################################

test: test-unit
test-all: test-unit test-ledger-mock test-race test-cover

test-unit:
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./...

test-race:
	@VERSION=$(VERSION) go test -mod=readonly -race -tags='ledger test_ledger_mock' ./...

test-cover:
	@go test -mod=readonly -timeout 30m -race -coverprofile=coverage.txt -covermode=atomic -tags='ledger test_ledger_mock' ./...

test-ledger-mock:
	@go test -mod=readonly -tags='ledger test_ledger_mock' ./...

test-ledger: test-ledger-mock
	@go test -mod=readonly -tags='ledger' ./...

test-property:
	@echo "Running property-based tests..."
	@go test -mod=readonly -tags='property' -run TestProperty ./...

benchmark:
	@go test -mod=readonly -bench=. ./...

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	@echo "--> Running linter"
	@golangci-lint run
	@go mod verify

format:
	@go fmt ./...

###############################################################################
###                                Localnet                                 ###
###############################################################################

localnet-init: localnet-clean localnet-build

localnet-build:
	$(MAKE) -C networks/local

localnet-start: localnet-stop
	@if ! [ -f build/node0/interbank-nettingd/config/genesis.json ]; then docker run --rm -v $(CURDIR)/build:/interbank-netting:Z interbank-netting/node "./setup.sh"; fi
	docker-compose up -d

localnet-stop:
	docker-compose down

localnet-clean:
	docker-compose down
	sudo rm -rf build/*

###############################################################################
###                                 Docker                                  ###
###############################################################################

docker-build:
	@docker build -t interbank-netting/interbank-nettingd .

docker-run:
	@docker run -it --rm interbank-netting/interbank-nettingd

###############################################################################
###                              Documentation                              ###
###############################################################################

update-swagger-docs: statik
	$(BINDIR)/statik -src=client/docs/swagger-ui -dest=client/docs -f -m
	@if [ -n "$(git status --porcelain)" ]; then \
        echo "\033[91mSwagger docs are out of sync!!!\033[0m";\
        exit 1;\
    else \
        echo "\033[92mSwagger docs are in sync\033[0m";\
    fi

godocs:
	@echo "--> Wait a few seconds and visit http://localhost:6060/pkg/github.com/interbank-netting/cosmos"
	godoc -http=:6060

###############################################################################
###                                 Clean                                   ###
###############################################################################

clean:
	rm -rf build/

.PHONY: all build-linux install format lint test test-all test-cover test-unit test-race test-property benchmark \
	localnet-init localnet-build localnet-start localnet-stop localnet-clean \
	docker-build docker-run \
	update-swagger-docs godocs clean