# Makefile for PipeGen

# Project information
PROJECT_NAME := pipegen
MAIN_PACKAGE := .
BIN_DIR := bin
DIST_DIR := dist

# Go build settings
GO := go
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
CGO_ENABLED := 0

# Version info (dynamic from git or fallback)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# LDFLAGS for version info
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -w -s"

# Platform targets for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/386

.PHONY: all build clean test lint fmt help install dev deps build-all release

## Default target
all: clean fmt lint test build

## Build binary for current platform
build:
	@echo "Building $(PROJECT_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(PROJECT_NAME) $(MAIN_PACKAGE)
	@echo "✓ Built $(BIN_DIR)/$(PROJECT_NAME)"

## Build for all platforms
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(PROJECT_NAME); \
		if [ $$os = "windows" ]; then output_name=$(PROJECT_NAME).exe; fi; \
		echo "Building for $$os/$$arch..."; \
		mkdir -p $(DIST_DIR)/$(PROJECT_NAME)-$$os-$$arch; \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=$$arch \
		$(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(PROJECT_NAME)-$$os-$$arch/$$output_name $(MAIN_PACKAGE); \
		if [ $$? -eq 0 ]; then \
			if [ $$os = "windows" ]; then \
				(cd $(DIST_DIR) && zip -q $(PROJECT_NAME)-$$os-$$arch.zip $(PROJECT_NAME)-$$os-$$arch/*); \
			else \
				(cd $(DIST_DIR) && tar -czf $(PROJECT_NAME)-$$os-$$arch.tar.gz $(PROJECT_NAME)-$$os-$$arch/); \
			fi; \
			rm -rf $(DIST_DIR)/$(PROJECT_NAME)-$$os-$$arch; \
		else \
			echo "Failed to build for $$os/$$arch"; \
		fi; \
	done
	@echo "✓ Built all platforms in $(DIST_DIR)/"
## Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(DIST_DIR)
	@$(GO) clean
	@echo "✓ Cleaned"

## Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests passed"

## Test with coverage report
test-coverage: test
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

## Lint code
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, installing..."; \
		$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi
	@echo "✓ Linting completed"

## Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Code formatted"

## Lint code with fixes
lint-fix:
	@echo "Running linters with auto-fixes..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	else \
		echo "golangci-lint not found, installing..."; \
		$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --fix; \
	fi
	@echo "✓ Linting with fixes completed"

## Pre-commit checks (runs before commits)
pre-commit: fmt lint test
	@echo "✅ All pre-commit checks passed!"

## Pre-push checks (comprehensive checks before pushing to main)
pre-push: clean fmt lint test build
	@echo "🚀 Pre-push checks completed successfully!"

## Quality gate (all quality checks)
quality: pre-push
	@echo "🎉 All quality gates passed!"

## Setup git hooks
setup-hooks:
	@echo "Setting up git hooks..."
	@chmod +x .githooks/pre-commit .githooks/pre-push
	@git config core.hooksPath .githooks
	@echo "✓ Git hooks configured"

## Install pre-commit framework
setup-pre-commit:
	@echo "Setting up pre-commit framework..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "✓ pre-commit hooks installed"; \
	else \
		echo "⚠️  pre-commit not found. Install with: pip install pre-commit"; \
	fi

## Install binary to GOBIN
install: build
	@echo "Installing $(PROJECT_NAME)..."
	@$(GO) install $(LDFLAGS) $(MAIN_PACKAGE)
	@echo "✓ Installed $(PROJECT_NAME) to $(shell go env GOBIN)"

## Development setup
dev: deps
	@echo "Setting up development environment..."
	@$(GO) mod download
	@$(GO) mod verify
	@echo "✓ Development environment ready"

## Install dependencies
deps:
	@echo "Installing dependencies..."
	@$(GO) mod tidy
	@echo "✓ Dependencies installed"

## Create release (requires git tag)
release:
	@if [ -z "$(shell git tag --points-at HEAD)" ]; then \
		echo "Error: No git tag found at HEAD. Please create a tag first."; \
		echo "Example: git tag v1.0.0 && git push origin v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release for $(VERSION)..."
	@$(MAKE) clean build-all
	@echo "✓ Release artifacts created in $(DIST_DIR)/"
	@echo "📦 Upload these files to GitHub release:"
	@ls -la $(DIST_DIR)/

## Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

## Run example pipeline
run-example: build
	@echo "Running example pipeline..."
	@./$(BIN_DIR)/$(PROJECT_NAME) init example-pipeline --ai --description "e-commerce analytics pipeline"
	@cd example-pipeline && ../$(BIN_DIR)/$(PROJECT_NAME) dashboard &
	@echo "✓ Example running. Visit http://localhost:8080"

## Help
help:
	@echo "PipeGen Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Build Targets:"
	@echo "  all           Clean, format, lint, test, and build"
	@echo "  build         Build binary for current platform"
	@echo "  build-all     Build for all supported platforms"
	@echo "  clean         Clean build artifacts"
	@echo ""
	@echo "Development Targets:"
	@echo "  fmt           Format code"
	@echo "  lint          Run linters"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  dev           Set up development environment"
	@echo "  deps          Install/update dependencies"
	@echo ""
	@echo "Distribution Targets:"
	@echo "  install       Install binary to GOBIN"
	@echo "  release       Create release artifacts"
	@echo "  version       Show version information"
	@echo ""
	@echo "Example Targets:"
	@echo "  run-example   Create and run example pipeline"
