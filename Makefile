# Antimoji Makefile
# High-performance emoji detection and removal CLI tool

.PHONY: help build test test-coverage benchmark lint security-scan clean install deps check-all release

# Default target
help: ## Show this help message
	@echo "Antimoji - High-performance emoji detection CLI"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)

# Core targets
build: ## Build the antimoji binary
	@echo "Building antimoji..."
	@if [ "$(OS)" = "Windows_NT" ]; then \
		go build -ldflags="$(LDFLAGS)" -o bin/antimoji.exe ./cmd/antimoji; \
	else \
		go build -ldflags="$(LDFLAGS)" -o bin/antimoji ./cmd/antimoji; \
	fi

build-release: ## Build optimized release binary
	@echo "Building optimized release binary..."
	CGO_ENABLED=0 go build -ldflags="-s -w $(LDFLAGS)" -o bin/antimoji ./cmd/antimoji

# Dependencies
deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	go mod tidy

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/goreleaser/goreleaser@latest

# Testing
test: ## Run all tests
	@echo "Running tests..."
	go test -race -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | tail -1

test-coverage-check: ## Check if test coverage meets minimum requirement (79%)
	@echo "Checking test coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./... > /dev/null
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $${COVERAGE}%"; \
	if [ "$$(echo "$$COVERAGE < 79" | bc -l)" = "1" ]; then \
		echo "ERROR: Test coverage $${COVERAGE}% is below required 79%"; \
		exit 1; \
	else \
		echo "Test coverage $${COVERAGE}% meets requirement"; \
	fi

benchmark: ## Run benchmark tests
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem -run=^$$ ./... | tee benchmark_results.txt
	@echo "Benchmark results saved to benchmark_results.txt"

# Quality checks
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run --timeout=5m

antimoji-lint: build ## Run antimoji on its own codebase
	@echo "Running antimoji linter..."
	@if [ -f ".antimoji.yaml" ]; then \
		./bin/antimoji scan --config=.antimoji.yaml --profile=zero-tolerance --threshold=300 --format=table .; \
	else \
		echo "WARNING: No .antimoji.yaml config found. Run 'make generate-allowlist' first."; \
		exit 1; \
	fi

security-scan: ## Run security scanner
	@echo "Running security scan..."
	gosec -quiet ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Allowlist management
generate-allowlist: build ## Generate antimoji allowlist configuration
	@echo "Generating antimoji allowlist configuration..."
	./bin/antimoji generate --type=ci-lint --output=.antimoji.yaml .
	@echo "Generated .antimoji.yaml configuration"

# Pre-commit integration
install-pre-commit: ## Install pre-commit framework
	@echo "Installing pre-commit framework..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "pre-commit already installed"; \
	elif command -v brew >/dev/null 2>&1; then \
		brew install pre-commit; \
	else \
		echo "ERROR: Please install pre-commit manually: brew install pre-commit"; \
		exit 1; \
	fi
	@pre-commit install
	@echo "Pre-commit hooks installed"

# Comprehensive quality check
check-all: deps fmt vet lint antimoji-lint security-scan test-coverage-check ## Run all quality checks
	@echo "All quality checks passed!"

# Installation
install: build ## Install antimoji to system location
	@echo "Installing antimoji..."
	sudo cp bin/antimoji /usr/local/bin/antimoji
	@echo "Installed to /usr/local/bin/antimoji"

uninstall: ## Uninstall antimoji from system
	@echo "Uninstalling antimoji..."
	sudo rm -f /usr/local/bin/antimoji
	@echo "Removed /usr/local/bin/antimoji"

# Clean up
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/ dist/
	rm -f coverage.out coverage.html
	rm -f *.backup.*

# Development setup
dev-setup: deps install-tools install-pre-commit generate-allowlist ## Set up complete development environment
	@echo "Development environment ready!"
	@echo "Pre-commit auto-cleaning enabled"
	@echo "Antimoji configuration generated"

# Release management
release-prepare: ## Prepare for release (VERSION required)
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release-prepare VERSION=v0.10.0"; \
		exit 1; \
	fi
	@echo "Preparing release $(VERSION)..."
	@make check-all
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Release $(VERSION) prepared. Push tag: git push origin $(VERSION)"

release-check: check-all build-release ## Run all checks before release
	@echo "Release checks passed!"
	@echo "Binary size: $$(du -h bin/antimoji | cut -f1)"

# CI targets
ci-test: ## CI test target
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

ci-lint: ## CI lint target
	golangci-lint run --out-format=github-actions --timeout=5m

# Help target (default)
.DEFAULT_GOAL := help
