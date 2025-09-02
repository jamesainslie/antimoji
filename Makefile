# Antimoji Makefile
# High-performance emoji detection and removal CLI tool

.PHONY: help build test test-coverage test-watch lint security-scan benchmark clean install deps check-all release-prepare release-check

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

# LD flags for build information
LDFLAGS = -X main.version=$(VERSION) \
          -X main.buildTime=$(BUILD_TIME) \
          -X main.gitCommit=$(GIT_COMMIT)

# Build targets
build: ## Build the antimoji binary
	@echo "Building antimoji..."
	go build -ldflags="$(LDFLAGS)" -o bin/antimoji ./cmd/antimoji

build-release: ## Build optimized release binary
	@echo "Building optimized release binary..."
	CGO_ENABLED=0 go build -ldflags="-s -w $(LDFLAGS)" -o bin/antimoji ./cmd/antimoji

# Development targets
deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	go mod tidy

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/golang/mock/mockgen@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/goreleaser/goreleaser@latest

# Testing targets
test: ## Run all tests
	@echo "Running tests..."
	go test -race -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | tail -1

test-coverage-check: ## Check if test coverage meets minimum requirement (85%)
	@echo "Checking test coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./... > /dev/null
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $${COVERAGE}%"; \
	if [ "$$(echo "$$COVERAGE < 85" | bc -l)" = "1" ]; then \
		echo "❌ Test coverage $${COVERAGE}% is below required 85%"; \
		exit 1; \
	else \
		echo "✅ Test coverage $${COVERAGE}% meets requirement"; \
	fi

test-watch: ## Run tests in watch mode (requires entr)
	@echo "Running tests in watch mode (Ctrl+C to stop)..."
	find . -name "*.go" | entr -c go test -race ./...

test-benchmark: ## Run benchmark tests
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem -run=^$$ ./...

test-benchmark-compare: ## Compare benchmark results (requires BEFORE and AFTER files)
	@echo "Comparing benchmark results..."
	@if [ ! -f "$(BEFORE)" ] || [ ! -f "$(AFTER)" ]; then \
		echo "Usage: make test-benchmark-compare BEFORE=before.txt AFTER=after.txt"; \
		exit 1; \
	fi
	benchcmp $(BEFORE) $(AFTER)

# Quality targets
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run --timeout=5m

lint-fix: ## Run linter with auto-fix
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix --timeout=5m

security-scan: ## Run security scanner
	@echo "Running security scan..."
	gosec -quiet ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Performance targets
benchmark: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./... | tee benchmark_results.txt

profile-cpu: ## Generate CPU profile
	@echo "Generating CPU profile..."
	go test -bench=BenchmarkDetectEmojis -cpuprofile=cpu.prof ./internal/core/detector
	@echo "View with: go tool pprof cpu.prof"

profile-mem: ## Generate memory profile
	@echo "Generating memory profile..."
	go test -bench=BenchmarkDetectEmojis -memprofile=mem.prof ./internal/core/detector
	@echo "View with: go tool pprof mem.prof"

profile-trace: ## Generate execution trace
	@echo "Generating execution trace..."
	go test -bench=BenchmarkDetectEmojis -trace=trace.out ./internal/core/detector
	@echo "View with: go tool trace trace.out"

# Comprehensive quality check
check-all: deps fmt vet lint security-scan test-coverage-check ## Run all quality checks
	@echo "✅ All quality checks passed!"

# Mock generation
generate-mocks: ## Generate mocks for interfaces
	@echo "Generating mocks..."
	go generate ./...

# Documentation
docs-serve: ## Serve documentation locally
	@echo "Serving documentation on http://localhost:6060"
	godoc -http=:6060

# Installation targets
install: build ## Install antimoji to system location
	@echo "Installing antimoji..."
	@if [ "$(shell uname)" = "Darwin" ]; then \
		sudo cp bin/antimoji /usr/local/bin/antimoji; \
		echo "Installed to /usr/local/bin/antimoji"; \
	elif [ "$(shell uname)" = "Linux" ]; then \
		sudo cp bin/antimoji /usr/local/bin/antimoji; \
		echo "Installed to /usr/local/bin/antimoji"; \
	elif [ "$(shell uname | grep -i cygwin)" ]; then \
		cp bin/antimoji.exe /usr/local/bin/antimoji.exe; \
		echo "Installed to /usr/local/bin/antimoji.exe"; \
	else \
		echo "Unsupported platform: $(shell uname)"; \
		echo "Please manually copy bin/antimoji to your PATH"; \
		exit 1; \
	fi

uninstall: ## Uninstall antimoji from system
	@echo "Uninstalling antimoji..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname)" = "Linux" ]; then \
		sudo rm -f /usr/local/bin/antimoji; \
		echo "Removed /usr/local/bin/antimoji"; \
	elif [ "$(shell uname | grep -i cygwin)" ]; then \
		rm -f /usr/local/bin/antimoji.exe; \
		echo "Removed /usr/local/bin/antimoji.exe"; \
	else \
		echo "Please manually remove antimoji from your PATH"; \
	fi

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof trace.out
	rm -f benchmark_results.txt

clean-deps: ## Clean dependency cache
	@echo "Cleaning dependency cache..."
	go clean -modcache

# Release targets
release-prepare: ## Prepare for release (VERSION required)
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release-prepare VERSION=v0.1.0"; \
		exit 1; \
	fi
	@echo "Preparing release $(VERSION)..."
	@echo "Running pre-release checks..."
	@make check-all
	@echo "Creating git tag..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Release $(VERSION) prepared. Push tag to trigger release: git push origin $(VERSION)"

release-check: check-all build-release ## Run all checks before release
	@echo "✅ Release checks passed!"
	@echo "Binary size: $$(du -h bin/antimoji | cut -f1)"
	@echo "Ready for release!"

release-snapshot: ## Create snapshot release with GoReleaser
	@echo "Creating snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "GoReleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

release-local: ## Test release locally with GoReleaser
	@echo "Testing release locally..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
		goreleaser build --snapshot --clean; \
	else \
		echo "GoReleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

# Development workflow
dev-setup: deps install-tools ## Set up development environment
	@echo "✅ Development environment set up!"
	@echo "Run 'make test-watch' to start TDD workflow"

# CI/CD helpers
ci-test: ## CI test target
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

ci-lint: ## CI lint target  
	golangci-lint run --out-format=github-actions --timeout=5m

ci-security: ## CI security scan target
	gosec -fmt=sarif -out=gosec.sarif ./...

ci-build-all: ## Build all platform binaries for CI
	@echo "Building all platform binaries..."
	@mkdir -p dist
	@for goos in linux darwin windows; do \
		for goarch in amd64 arm64; do \
			if [ "$$goos" = "windows" ] && [ "$$goarch" = "arm64" ]; then \
				continue; \
			fi; \
			binary="dist/antimoji_$${goos}_$${goarch}"; \
			if [ "$$goos" = "windows" ]; then \
				binary="$${binary}.exe"; \
			fi; \
			echo "Building $$goos/$$goarch..."; \
			GOOS=$$goos GOARCH=$$goarch CGO_ENABLED=0 \
			go build -ldflags="$(LDFLAGS)" -o "$$binary" ./cmd/antimoji; \
		done; \
	done
	@echo "✅ All platform binaries built successfully"

ci-test-binaries: ci-build-all ## Test all built binaries
	@echo "Testing built binaries..."
	@for binary in dist/antimoji_*; do \
		if [[ "$$binary" == *"windows"* ]]; then \
			echo "⏭️ Skipping Windows binary: $$binary"; \
		else \
			echo "Testing: $$binary"; \
			chmod +x "$$binary"; \
			"$$binary" version || echo "❌ Binary test failed: $$binary"; \
		fi; \
	done

ci-package: ci-build-all ## Create distribution packages
	@echo "Creating distribution packages..."
	@for binary in dist/antimoji_*; do \
		platform=$$(echo "$$binary" | cut -d'_' -f3); \
		arch=$$(echo "$$binary" | cut -d'_' -f4 | cut -d'.' -f1); \
		if [[ "$$binary" == *"windows"* ]]; then \
			zip -j "dist/antimoji_$${platform}_$${arch}.zip" "$$binary" README.md LICENSE CHANGELOG.md; \
		else \
			tar -czf "dist/antimoji_$${platform}_$${arch}.tar.gz" -C dist "$$(basename $$binary)" -C .. README.md LICENSE CHANGELOG.md; \
		fi; \
	done
	@echo "✅ Distribution packages created"

# Docker targets (if needed)
docker-build: ## Build Docker image
	docker build -t antimoji:latest .

docker-test: ## Run tests in Docker
	docker run --rm -v $(PWD):/app -w /app golang:1.22 make test

# Version information
version: ## Show version information
	@echo "Go version: $$(go version)"
	@echo "Git commit: $$(git rev-parse --short HEAD)"
	@echo "Build time: $$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Help target (default)
.DEFAULT_GOAL := help
