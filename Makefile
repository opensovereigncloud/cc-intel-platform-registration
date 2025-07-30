# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Docker image parameters
BINARY_NAME=cc-intel-platform-registration
VERSION ?=dev
CONTAINER_TOOL ?= docker
IMG_REGISTRY ?= local
GO_MOD_VERSION ?= 1.24.5
IMAGE_BUILD_ARGS ?= --build-arg GO_MOD_VERSION=$(GO_MOD_VERSION) --build-arg TARGET_VERSION=${VERSION}
IMAGE=$(IMG_REGISTRY)/cc-intel-platform-registration:$(VERSION)
CONTAINER_DEV_RUN = $(CONTAINER_TOOL) run --rm -e LD_LIBRARY_PATH=/cc_build_dir/build/lib $(IMAGE)

.PHONY: build-image
build-image:
	$(CONTAINER_TOOL) build -t $(IMAGE) $(IMAGE_BUILD_ARGS) .

.PHONY: image-push
image-push:
	$(CONTAINER_TOOL) push $(IMAGE)

clean:
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
	cd $(PWD)/third_party/mp_management && $(MAKE) clean
	cd $(PWD)/third_party/sgx_platform_info && $(MAKE) clean

mp_management:
	cd $(PWD)/third_party/mp_management && $(MAKE) 

sgx_platform_info:
	cd $(PWD)/third_party/sgx_platform_info && $(MAKE) 

deps:
	$(GOMOD) download
	$(GOMOD) tidy

test: 
	$(GOTEST) -v ./...

.PHONY: vuln-check
vuln-check: govulncheck
	$(GOVULNCHECK) ./...

.PHONY: lint-check
lint-check: golangci-lint     ## Run golangci-lint linter
	$(GOLANGCI_LINT) run ./...

.PHONY: deadcode-check
deadcode-check: deadcode 
	$(DEADCODE) ./...

# # Combined check target that runs all verifications
.PHONY: check
check: deps vuln-check lint-check deadcode-check test


# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: lint build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: build-dev-image
build-dev-image: IMAGE_BUILD_ARGS := --target builder --build-arg GO_MOD_VERSION=$(GO_MOD_VERSION) --build-arg TARGET_VERSION=dev
build-dev-image: build-image

# Generic target to run any make command inside Docker
.PHONY: docker-%
docker-%: build-dev-image
	$(CONTAINER_DEV_RUN) make $*

##@ Build
.PHONY: build ## Build manager binary.
build: mp_management sgx_platform_info  deps fmt vet 
	$(GOBUILD) -v \
	-ldflags "-s -w -X 'main.version=$(VERSION)' -X 'main.buildDate=$(shell date)'" \
	-o $(BINARY_NAME) \
	-trimpath

ifndef ignore-not-found
  ignore-not-found = false
endif


##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOVULNCHECK = $(LOCALBIN)/govulncheck
DEADCODE = $(LOCALBIN)/deadcode

## Tool Versions
GOLANGCI_LINT_VERSION ?= v1.63.4


.PHONY: deadcode
deadcode: $(DEADCODE) ## Download deadcode locally if necessary.
$(DEADCODE): $(LOCALBIN)
	$(call go-install-tool,$(DEADCODE),golang.org/x/tools/cmd/deadcode,latest)

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK) ## Download govulncheck locally if necessary.
$(GOVULNCHECK): $(LOCALBIN)
	$(call go-install-tool,$(GOVULNCHECK),golang.org/x/vuln/cmd/govulncheck,latest)


.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
