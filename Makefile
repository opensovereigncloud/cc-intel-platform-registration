# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOPATH=$(shell go env GOPATH)

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


BINARY_NAME=cc-intel-platform-registration

GOVULNCHECK=$(GOCMD) run golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: all build test clean deps lint security-check check

all: check build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./...

clean:
	$(GOCMD) clean
	rm -f $(BINARY_NAME)

deps:
	$(GOMOD) download
	$(GOMOD) tidy

.PHONY: lint
lint: golangci-lint 
	$(GOLANGCI_LINT) run ./...
	$(GOCMD) vet ./...

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
golangci-lint:
	@[ -f $(GOLANGCI_LINT) ] || { \
		set -e ;\
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) v1.63.4 ;\
	}

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

deadcode:
	@echo "Checking for dead code..."
	$(GOCMD) run golang.org/x/tools/cmd/deadcode@latest ./...

security-check:
	$(GOVULNCHECK) ./...

# Combined check target that runs all verifications
check: deps lint deadcode security-check test