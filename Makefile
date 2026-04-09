# Makefile for Verda Terraform Provider
#
# Usage:
#   make help          - Show this help
#   make build         - Build the provider
#   make install       - Install provider locally for testing
#   make test          - Run unit tests
#   make test-integration - Run integration tests
#   make pre-commit    - Run all configured pre-commit hooks
#   make lint          - Run Go linting
#   make security      - Run security scan (gitleaks + gosec)
#   make fmt           - Format Go and Terraform code
#   make clean         - Clean build artifacts
#

.PHONY: help build install test test-integration pre-commit lint security fmt clean clean-test release

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME := terraform-provider-verda
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO := go
GOFMT := gofmt
GOFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Colors
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

# Help target
help:
	@echo ""
	@echo "$(CYAN)Verda Terraform Provider$(RESET)"
	@echo ""
	@echo "$(GREEN)Build & Install:$(RESET)"
	@echo "  make build           Build the provider binary"
	@echo "  make install         Install provider locally for testing"
	@echo ""
	@echo "$(GREEN)Testing:$(RESET)"
	@echo "  make test            Run unit tests"
	@echo "  make test-integration Run integration tests (requires .env with credentials)"
	@echo ""
	@echo "$(GREEN)Code Quality:$(RESET)"
	@echo "  make pre-commit      Run all configured pre-commit hooks"
	@echo "  make lint            Run Go linting (golangci-lint)"
	@echo "  make security        Run security scans (gitleaks + gosec)"
	@echo "  make fmt             Format Go and Terraform code"
	@echo ""
	@echo "$(GREEN)Maintenance:$(RESET)"
	@echo "  make clean           Clean build artifacts"
	@echo "  make deps            Download dependencies"
	@echo ""
	@echo "$(GREEN)Release:$(RESET)"
	@echo "  make release VERSION=vX.Y.Z  Prepare a new release (updates CHANGELOG.md)"
	@echo ""

# Build the provider
build:
	@echo "$(CYAN)Building $(BINARY_NAME)...$(RESET)"
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .
	@echo "$(GREEN)Build complete: $(BINARY_NAME)$(RESET)"

# Install provider locally for testing
install: build
	@echo "$(CYAN)Installing provider locally...$(RESET)"
	@mkdir -p ~/.terraform.d/plugins/verda-cloud/verda/99.0.0/$$($(GO) env GOOS)_$$($(GO) env GOARCH)
	@cp $(BINARY_NAME) ~/.terraform.d/plugins/verda-cloud/verda/99.0.0/$$($(GO) env GOOS)_$$($(GO) env GOARCH)/
	@echo "$(GREEN)Provider installed to ~/.terraform.d/plugins/verda-cloud/verda/99.0.0/$(RESET)"

# Run unit tests
test:
	@echo "$(CYAN)Running unit tests...$(RESET)"
	$(GO) test -v -short ./...

# Run integration tests
test-integration:
	@echo "$(CYAN)Running integration tests...$(RESET)"
	@if [ -f .env ]; then \
		echo "$(YELLOW)Loading environment from .env...$(RESET)"; \
		set -a && . ./.env && set +a && ./scripts/run-integration-tests.sh; \
	else \
		echo "$(YELLOW)No .env file found, running without additional env vars...$(RESET)"; \
		./scripts/run-integration-tests.sh; \
	fi

# Run all configured pre-commit hooks
pre-commit:
	@echo "$(CYAN)Running pre-commit hooks...$(RESET)"
	@command -v pre-commit >/dev/null 2>&1 || (echo "$(RED)pre-commit is not installed. Install it with pipx/pip/brew and rerun.$(RESET)" && exit 1)
	pre-commit run --all-files

# Run Go linting only
lint:
	@echo "$(CYAN)Running golangci-lint...$(RESET)"
	golangci-lint run --timeout=5m

# Run security scans
security:
	@echo "$(CYAN)Running security scans...$(RESET)"
	@echo ""
	@echo "$(YELLOW)Gitleaks (secret detection):$(RESET)"
	gitleaks detect --source . --config .gitleaks.toml --verbose
	@echo ""
	@echo "$(YELLOW)Gosec (Go security scanner):$(RESET)"
	gosec -exclude-dir=test ./...

# Format code
fmt:
	@echo "$(CYAN)Formatting code...$(RESET)"
	@echo "$(YELLOW)Formatting Go code...$(RESET)"
	$(GOFMT) -w .
	$(GO) mod tidy
	@echo "$(YELLOW)Formatting Terraform code...$(RESET)"
	terraform fmt -recursive examples/ test/ 2>/dev/null || true
	@echo "$(GREEN)Formatting complete$(RESET)"

# Download dependencies
deps:
	@echo "$(CYAN)Downloading dependencies...$(RESET)"
	$(GO) mod download
	$(GO) mod tidy
	@echo "$(GREEN)Dependencies updated$(RESET)"

# Clean build artifacts
clean:
	@echo "$(CYAN)Cleaning build artifacts...$(RESET)"
	rm -f $(BINARY_NAME)
	rm -rf dist/
	$(GO) clean
	@echo "$(GREEN)Clean complete$(RESET)"

# Clean test resources
clean-test:
	@echo "$(CYAN)Cleaning test resources...$(RESET)"
	./scripts/run-integration-tests.sh --cleanup-only

# Release management
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Error: VERSION is required$(RESET)"; \
		echo "Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "$(CYAN)Preparing release $(VERSION)...$(RESET)"
	@./scripts/release.sh $(VERSION)
	@echo "$(GREEN)Release $(VERSION) prepared. Review CHANGELOG.md and commit.$(RESET)"
