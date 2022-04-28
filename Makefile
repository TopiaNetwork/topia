#!/usr/bin/make -f

BUILDDIR ?= $(CURDIR)/build

BUILD_TAGS?=topia

# If building a release, please checkout the version tag to get the correct version setting
ifneq ($(shell git symbolic-ref -q --short HEAD),)
VERSION := unreleased-$(shell git symbolic-ref -q --short HEAD)-$(shell git rev-parse HEAD)
else
VERSION := $(shell git describe)
endif

LD_FLAGS = -X github.com/TopiaNetwork/topia/version.TPVersionDefault=$(VERSION)
BUILD_FLAGS = -mod=readonly -ldflags "$(LD_FLAGS)"
CGO_ENABLED ?= 1

# handle badgerdb
ifeq (badgerdb,$(findstring badgerdb,$(TOPIA_BUILD_OPTIONS)))
  BUILD_TAGS += badgerdb
endif

# handle rocksdb
ifeq (rocksdb,$(findstring rocksdb,$(TOPIA_BUILD_OPTIONS)))
  CGO_ENABLED=1
  BUILD_TAGS += rocksdb
endif

# allow users to pass additional flags via the conventional LDFLAGS variable
LD_FLAGS += $(LDFLAGS)

all: check build install
.PHONY: all

###############################################################################
###                                Build Topia                         ###
###############################################################################

build: $(BUILDDIR)/
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -tags '$(BUILD_TAGS)' -o $(BUILDDIR)/ ./runner/topia
.PHONY: build

install:
	CGO_ENABLED=$(CGO_ENABLED) go install $(BUILD_FLAGS) -tags $(BUILD_TAGS) ./runner/topia
.PHONY: install

$(BUILDDIR)/:
	mkdir -p $@

###############################################################################
###                  Testing                       ###
###############################################################################

.PHONY: test
test:
	# test all packages
	GO111MODULE=on go test `go list ./... | grep -v -e github.com/TopiaNetwork/topia/common -e github.com/TopiaNetwork/topia/ledger/backend`

###############################################################################
###                  Formatting, linting, and vetting                       ###
###############################################################################

format:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/client9/misspell/cmd/misspell
	find . -name '*.go' -type f -not -path "*.git*" -not -name '*.pb.go' -not -name '*pb_test.go' | xargs gofmt -w -s
	find . -name '*.go' -type f -not -path "*.git*" -not -name '*.pb.go' -not -name '*pb_test.go' | xargs misspell -w
	find . -name '*.go' -type f -not -path "*.git*"  -not -name '*.pb.go' -not -name '*pb_test.go' | xargs goimports -w -local github.com/TopiaNetwork/topia
.PHONY: format

lint:
	@echo "--> Running linter"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run
.PHONY: lint

DESTINATION = ./index.html.md


clean:
	rm -rf $(CURDIR)/artifacts/ $(BUILDDIR)/
