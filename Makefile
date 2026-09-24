GO ?= go
GOFMT ?= gofmt
GOTOOLCHAIN_VERSION := go1.27.1
VERSION ?= dev
GORELEASER_VERSION := v2.18.2
GOVULNCHECK_VERSION := v1.1.4
GOLANGCI_LINT_VERSION := v2.14.0
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: all build install test vet lint fmt-check check vuln release-check snapshot clean
all: check build
build:
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/ssmsh .
install:
	$(GO) install -trimpath -ldflags '$(LDFLAGS)' .
test:
	$(GO) test -race ./...
vet:
	$(GO) vet ./...
lint:
	GOTOOLCHAIN=$(GOTOOLCHAIN_VERSION) $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run
fmt-check:
	@test -z "$$($(GOFMT) -l .)" || ($(GOFMT) -l .; echo 'Run gofmt on the files above'; exit 1)
check: fmt-check vet test
vuln:
	GOTOOLCHAIN=$(GOTOOLCHAIN_VERSION) $(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...
release-check:
	GOTOOLCHAIN=$(GOTOOLCHAIN_VERSION) $(GO) run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) check
snapshot:
	GOTOOLCHAIN=$(GOTOOLCHAIN_VERSION) $(GO) run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) release --snapshot --clean
clean:
	rm -rf bin dist
