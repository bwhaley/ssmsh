SHELL := /bin/bash
PROJECT := github.com/bwhaley/ssmsh
PKGS := $(shell go list ./... | grep -v /vendor)
EXECUTABLE := ssmsh
PKG := ssmsh
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)

.PHONY: build test golint docs $(PROJECT) $(PKGS) vendor

VERSION := $(shell echo ${SSMSH_VERSION})
ifeq "$(VERSION)" ""
    $(error must define SSMSH_VERSION env var)
endif

GOVERSION := $(shell go version | grep 1.15)
ifeq "$(GOVERSION)" ""
    $(error must be running Go version 1.15.x)
endif

ifndef $(GOPATH)
   GOPATH=$(shell go env GOPATH)
   export GOPATH
endif

all: test build

FGT := $(GOPATH)/bin/fgt
$(FGT):
	go get github.com/GeertJohan/fgt

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get golang.org/x/lint

DEP := $(GOPATH)/bin/dep
$(DEP):
	go get -u github.com/golang/dep

GO_LDFLAGS := -X $(shell go list ./$(PACKAGE)).GitCommit=$(GIT_COMMIT) -X main.Version=${VERSION}

test: $(PKGS)

$(PKGS): $(GOLINT)
	@echo "FORMATTING"
	go fmt $@
	@echo "LINTING"
	golint $@
	@echo "Vetting"
	go vet -v $@
	@echo "TESTING"
	go test -v $@

vendor: $(DEP)
	$(DEP) ensure

build:
	go build -i -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE) $(PROJECT)
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE)-linux-amd64
build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE)-darwin-amd64

clean:
	rm -f $(GOPATH)/bin/$(EXECUTABLE) $(GOPATH)/bin/$(EXECUTABLE)-*
