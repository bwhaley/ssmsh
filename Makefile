SHELL := /bin/bash
PROJECT := github.com/bwhaley/ssmsh
PKGS := $(shell go list ./... | grep -v /vendor)
EXECUTABLE := ssmsh
PKG := ssmsh
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)

.PHONY: build test golint docs $(PROJECT) $(PKGS) vendor

GOVERSION := $(shell go version | grep 1.12)
ifeq "$(GOVERSION)" ""
    $(error must be running Go version 1.12.x)
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
	GOOS=linux GOARCH=amd64 go build -o $(GOPATH)/bin/$(EXECUTABLE)-linux-amd64
build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o $(GOPATH)/bin/$(EXECUTABLE)-darwin-amd64

clean:
	rm -f $(GOPATH)/bin/$(EXECUTABLE) $(GOPATH)/bin/$(EXECUTABLE)-*
