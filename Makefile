# Go IP Whitelist by Country - Makefile

# Variables
BINARY_NAME=ip-whitelist
MAIN_PATH=./cmd/app
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
MODULE_NAME=github.com/anisimovdk/ip-whitelist-by-country

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION ?= $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS = -s -w \
	-X '$(MODULE_NAME)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE_NAME)/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X '$(MODULE_NAME)/internal/version.BuildDate=$(BUILD_DATE)' \
	-X '$(MODULE_NAME)/internal/version.GoVersion=$(GO_VERSION)'
TAG ?= latest

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Test targets
.PHONY: test
test: ## Run all tests
	@echo "Running all tests..."
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run all tests with verbose output
	@echo "Running all tests with verbose output..."
	go test -v ./...

.PHONY: test-cover
test-cover: ## Run all tests with coverage
	@echo "Running tests with coverage..."
	go test -cover ./...

.PHONY: test-cover-100
test-cover-100: ## Run tests and fail unless total coverage is 100%
	@echo "Running tests with coverage (minimum: 100%)..."
	@go test -coverprofile=coverage.out ./...
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/{gsub(/%/,"",$$NF); print $$NF}'); \
		echo "Total coverage: $${total}%"; \
		awk -v total="$${total}" 'BEGIN{exit !(total+0 >= 100.0)}' || (echo "Coverage $${total}% is below 100.0%"; exit 1)

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race ./...

# Build targets
.PHONY: build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PATH)

# Release targets
.PHONY: release-build
release-build: ## Build release binaries for all platforms
	@echo "Building release binaries..."
	@mkdir -p release
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o release/$(BINARY_NAME)-linux-amd64-$(VERSION) $(MAIN_PATH)
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o release/$(BINARY_NAME)-linux-arm64-$(VERSION) $(MAIN_PATH)
	@echo "Building for Linux ARMv7..."
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "$(LDFLAGS)" -o release/$(BINARY_NAME)-linux-armv7-$(VERSION) $(MAIN_PATH)
	@echo "Generating checksums..."
	@cd release && sha256sum * > checksums.txt
	@echo "Release binaries built successfully!"
	@ls -la release/

# Development targets
.PHONY: run
run: ## Run the application
	@echo "Running application..."
	go run $(MAIN_PATH)

.PHONY: run-dev
run-dev: ## Run the application with development settings
	@echo "Running application in development mode..."
	go run $(MAIN_PATH) --port=8080 --cache-duration=5m

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "Running golangci-lint..."
	golangci-lint run

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	go mod tidy

.PHONY: mod-download
mod-download: ## Download go modules
	@echo "Downloading modules..."
	go mod download

# Quality targets
.PHONY: check
check: fmt vet test ## Run format, vet, and tests

.PHONY: ci
ci: mod-tidy fmt vet test-verbose test-race test-cover ## Run all CI checks

.PHONY: release
release: release-build ## Prepare a complete release (test + build binaries)

.PHONY: clean-cache
clean-cache: ## Clean go build cache
	@echo "Cleaning build cache..."
	go clean -cache

# Clean targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux
	rm -f coverage.out
	go clean

.PHONY: clean-all
clean-all: clean clean-cache docker-clean ## Clean everything including Docker resources and releases

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GO_VERSION=$(GO_VERSION) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

.PHONY: docker-buildx-setup
docker-buildx-setup: ## Setup Docker buildx for multi-platform builds
	@echo "Setting up Docker buildx..."
	docker buildx create --name multiarch --use --bootstrap || true
	docker buildx inspect --bootstrap

.PHONY: docker-build-multiarch
docker-build-multiarch: docker-build-amd64 docker-build-arm64 docker-build-armv7 ## Build Docker images for multiple architectures

.PHONY: docker-build-amd64
docker-build-amd64: ## Build Docker image for AMD64 and load to local docker
	@echo "Building Docker image for linux/amd64..."
	docker buildx build --platform linux/amd64 -t $(BINARY_NAME):${TAG} --load .

.PHONY: docker-build-arm64
docker-build-arm64: ## Build Docker image for ARM64 and load to local docker
	@echo "Building Docker image for linux/arm64..."
	docker buildx build --platform linux/arm64 -t $(BINARY_NAME):${TAG} --load .

.PHONY: docker-build-armv7
docker-build-armv7: ## Build Docker image for linux/arm/v7 and load to local docker
	@echo "Building Docker image for linux/arm/v7..."
	docker buildx build --platform linux/arm/v7 -t $(BINARY_NAME):${TAG} --load .

.PHONY: docker-run
docker-run: ## Run Docker container in detached mode
	@echo "Running Docker container in detached mode..."
	docker run -d -p 8080:8080 --name $(BINARY_NAME) $(BINARY_NAME):${TAG}

.PHONY: docker-stop
docker-stop: ## Stop Docker container
	@echo "Stopping Docker container..."
	docker stop $(BINARY_NAME) || true
	docker stop $(BINARY_NAME)-dev || true

.PHONY: docker-clean
docker-clean: ## Remove Docker container and image
	@echo "Cleaning Docker resources..."
	docker stop $(BINARY_NAME) || true
	docker stop $(BINARY_NAME)-dev || true
	docker rm $(BINARY_NAME) || true
	docker rm $(BINARY_NAME)-dev || true
	docker rmi $(BINARY_NAME):${TAG} || true

# Benchmark targets
.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. ./...

.PHONY: bench-mem
bench-mem: ## Run benchmarks with memory stats
	@echo "Running benchmarks with memory stats..."
	go test -bench=. -benchmem ./...

# Security targets
.PHONY: sec-check
sec-check: ## Run security checks (requires gosec to be installed)
	@echo "Running security checks..."
	gosec ./...

# Installation targets
.PHONY: install
install: ## Install the application
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PATH)

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@${TAG}
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@${TAG}

# Info targets
.PHONY: version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

.PHONY: deps
deps: ## Show dependencies
	@echo "Dependencies:"
	@go list -m all

.PHONY: outdated
outdated: ## Check for outdated dependencies (requires go-mod-outdated)
	@echo "Checking for outdated dependencies..."
	go list -u -m all

.DEFAULT_GOAL := help
