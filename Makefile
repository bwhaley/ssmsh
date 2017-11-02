SHELL := /bin/bash
PROJECT := github.com/kountable/pssh
PKGS := $(shell go list ./... | grep -v /vendor)
EXECUTABLE := pssh
PKG := pssh
DOCKER_REGISTRY := jgeiger
DOCKER_IMAGE_NAME := example
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)

.PHONY: build test golint docs $(PROJECT) $(PKGS) vendor

GOVERSION := $(shell go version | grep 1.9)
ifeq "$(GOVERSION)" ""
    $(error must be running Go version 1.9)
endif

all: test build

FGT := $(GOPATH)/bin/fgt
$(FGT):
	go get github.com/GeertJohan/fgt

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get github.com/golang/lint/golint

DEP := $(GOPATH)/bin/dep
$(DEP):
	go get -u github.com/golang/dep

GO_LDFLAGS := -X $(shell go list ./$(PACKAGE)).GitCommit=$(GIT_COMMIT)

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
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -i -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE)-linux-amd64 $(PROJECT)
build-darwin:
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -a -installsuffix cgo -i -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE)-darwin-amd64 $(PROJECT)

clean:
	rm -f $(GOPATH)/bin/$(EXECUTABLE) $(GOPATH)/bin/$(EXECUTABLE)-*
