# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=cc-intel-platform-registration

# Docker image parameters
VERSION=dev
DOCKER_REGISTRY=local
DOCKER_IMAGE=$(DOCKER_REGISTRY)/cc-intel-platform-registration:$(VERSION)

# Tools
GOLANGCI_LINT=golangci-lint
STATICCHECK=staticcheck
GOVULNCHECK=$(GOCMD) run golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: all build test clean deps lint security-check check

all: check build-image

build-image:
	docker build -t $(DOCKER_IMAGE) .

push-image:
	docker push $(DOCKER_IMAGE)

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
	$(GOGET) -u honnef.co/go/tools/cmd/staticcheck
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.63.4

lint:
	$(GOLANGCI_LINT) run
	$(STATICCHECK) ./...
	$(GOCMD) vet ./...

deadcode:
	@echo "Checking for dead code..."
	$(GOCMD) run golang.org/x/tools/cmd/deadcode@latest ./...

security-check:
	$(GOVULNCHECK) ./...

# Combined check target that runs all verifications
check: deps lint deadcode security-check test