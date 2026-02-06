.DEFAULT_GOAL := help

CGO_ENABLED ?= 0
GO ?= go

BINARY_NAME := hyperfleet-cloud-provider
BUILD_DIR := bin
MAIN_PATH := cmd/provider/main.go

GIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY ?= $(shell git diff --quiet 2>/dev/null || echo "-dirty")
VERSION ?= $(GIT_SHA)$(GIT_DIRTY)
COMMIT ?= $(GIT_SHA)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

container_tool ?= podman

IMAGE_REGISTRY ?= quay.io/openshift-hyperfleet
IMAGE_NAME ?= hyperfleet-cloud-provider
IMAGE_TAG ?= latest

QUAY_USER ?=
DEV_TAG ?= dev-$(GIT_SHA)

LDFLAGS := -ldflags "-X github.com/openshift-hyperfleet/hyperfleet-cloud-provider/cmd/provider/version.Version=$(VERSION) -X github.com/openshift-hyperfleet/hyperfleet-cloud-provider/cmd/provider/version.Commit=$(COMMIT) -X github.com/openshift-hyperfleet/hyperfleet-cloud-provider/cmd/provider/version.BuildTime=$(BUILD_TIME)"

help:
	@echo ""
	@echo "HyperFleet Cloud Provider - Multi-cloud Kubernetes Token Provider"
	@echo ""
	@echo "make build                compile binary to bin/"
	@echo "make test                 run unit tests"
	@echo "make test-integration     run integration tests"
	@echo "make lint                 run golangci-lint"
	@echo "make fmt                  format code"
	@echo "make clean                delete build artifacts"
	@echo "make image                build container image"
	@echo "make image-push           build and push container image"
	@echo "make image-dev            build and push to personal Quay registry"
	@echo "make scan                 run security vulnerability scan"
.PHONY: help

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"
.PHONY: build

test:
	@echo "Running unit tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./pkg/... ./internal/... ./cmd/...
.PHONY: test

test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration -timeout=30m ./test/integration/...
.PHONY: test-integration

coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
.PHONY: coverage

lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run --config .golangci.yml
.PHONY: lint

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
.PHONY: fmt

vet:
	@echo "Running go vet..."
	$(GO) vet ./...
.PHONY: vet

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
.PHONY: clean

deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
.PHONY: deps

image:
	@echo "Building container image $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)..."
	$(container_tool) build \
		--platform linux/amd64 \
		--build-arg GIT_SHA=$(GIT_SHA) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "✅ Image built: $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)"
.PHONY: image

image-push: image
	@echo "Pushing image $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)..."
	$(container_tool) push $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	@echo "✅ Image pushed: $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)"
.PHONY: image-push

image-dev:
ifndef QUAY_USER
	@echo "❌ ERROR: QUAY_USER is not set"
	@echo ""
	@echo "Usage: QUAY_USER=myuser make image-dev"
	@echo ""
	@echo "This will build and push to: quay.io/$$QUAY_USER/$(IMAGE_NAME):$(DEV_TAG)"
	@exit 1
endif
	@echo "Building dev image quay.io/$(QUAY_USER)/$(IMAGE_NAME):$(DEV_TAG)..."
	$(container_tool) build \
		--platform linux/amd64 \
		--build-arg BASE_IMAGE=alpine:3.21 \
		--build-arg GIT_SHA=$(GIT_SHA) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t quay.io/$(QUAY_USER)/$(IMAGE_NAME):$(DEV_TAG) .
	@echo "Pushing dev image..."
	$(container_tool) push quay.io/$(QUAY_USER)/$(IMAGE_NAME):$(DEV_TAG)
	@echo ""
	@echo "✅ Dev image pushed: quay.io/$(QUAY_USER)/$(IMAGE_NAME):$(DEV_TAG)"
.PHONY: image-dev

image-test: image
	@echo "Testing container image..."
	$(container_tool) run --rm $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) version
	@echo "✅ Container image test passed"
.PHONY: image-test

scan:
	@echo "Running security vulnerability scan..."
	@which govulncheck > /dev/null || $(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
.PHONY: scan
