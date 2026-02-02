#!/usr/bin/env bash
#
# Integration Test Runner for Verda Terraform Provider
#
# This script runs integration tests against the Verda API using either
# Terraform or OpenTofu CLI. It includes automatic cleanup of test resources.
#
# Usage:
#   ./scripts/run-integration-tests.sh [options]
#
# Options:
#   --terraform     Use Terraform CLI (default)
#   --opentofu      Use OpenTofu CLI
#   --all           Run all tests including expensive GPU tests
#   --quick         Run only quick tests (skip instance, container, serverless_job)
#   --test NAME     Run a specific test (e.g., --test SSHKey)
#   --cleanup-only  Only run cleanup (no tests)
#   --help          Show this help message
#
# Environment Variables:
#   VERDA_CLIENT_ID      Required. Verda API client ID
#   VERDA_CLIENT_SECRET  Required. Verda API client secret
#   TF_CMD               Optional. Override terraform/tofu command path
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
TF_TOOL="terraform"
RUN_MODE="default"
SPECIFIC_TEST=""
CLEANUP_ONLY=false

# Function to print colored output
info() { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Show help
show_help() {
    head -32 "$0" | tail -30 | sed 's/^# //' | sed 's/^#//'
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --terraform)
            TF_TOOL="terraform"
            shift
            ;;
        --opentofu)
            TF_TOOL="tofu"
            shift
            ;;
        --all)
            RUN_MODE="all"
            shift
            ;;
        --quick)
            RUN_MODE="quick"
            shift
            ;;
        --test)
            SPECIFIC_TEST="$2"
            shift 2
            ;;
        --cleanup-only)
            CLEANUP_ONLY=true
            shift
            ;;
        --help|-h)
            show_help
            ;;
        *)
            error "Unknown option: $1"
            show_help
            ;;
    esac
done

# Check for required environment variables
check_env() {
    local missing=0

    if [[ -z "${VERDA_CLIENT_ID:-}" ]]; then
        error "VERDA_CLIENT_ID is not set"
        missing=1
    fi

    if [[ -z "${VERDA_CLIENT_SECRET:-}" ]]; then
        error "VERDA_CLIENT_SECRET is not set"
        missing=1
    fi

    if [[ $missing -eq 1 ]]; then
        echo ""
        info "Set environment variables or create a .env file:"
        echo "  export VERDA_CLIENT_ID='your-client-id'"
        echo "  export VERDA_CLIENT_SECRET='your-client-secret'"
        echo ""
        info "Or copy .env.example to .env and fill in your credentials"
        exit 1
    fi
}

# Load .env file if it exists
load_env() {
    local env_file="${PROJECT_ROOT}/.env"
    if [[ -f "$env_file" ]]; then
        info "Loading environment from .env file"
        set -a
        source "$env_file"
        set +a
    fi
}

# Check if terraform/tofu is installed
check_tf_tool() {
    if ! command -v "$TF_TOOL" &> /dev/null; then
        error "$TF_TOOL is not installed or not in PATH"
        echo ""
        if [[ "$TF_TOOL" == "terraform" ]]; then
            info "Install Terraform: https://www.terraform.io/downloads"
        else
            info "Install OpenTofu: https://opentofu.org/docs/intro/install/"
        fi
        exit 1
    fi

    local version
    version=$("$TF_TOOL" version -json 2>/dev/null | grep -o '"terraform_version":"[^"]*"' | cut -d'"' -f4 || "$TF_TOOL" version | head -1)
    info "Using $TF_TOOL version: $version"
}

# Build the provider
build_provider() {
    info "Building provider..."
    cd "$PROJECT_ROOT"
    go build -o terraform-provider-verda .
    success "Provider built successfully"
}

# Install provider locally for testing
install_provider_locally() {
    info "Installing provider locally for testing..."

    local os arch plugin_dir
    os=$(go env GOOS)
    arch=$(go env GOARCH)
    plugin_dir="${HOME}/.terraform.d/plugins/verda-cloud/verda/99.0.0/${os}_${arch}"

    mkdir -p "$plugin_dir"
    cp "${PROJECT_ROOT}/terraform-provider-verda" "$plugin_dir/"

    success "Provider installed to $plugin_dir"
}

# Run tests
run_tests() {
    info "Running integration tests with $TF_TOOL..."
    echo ""

    cd "$PROJECT_ROOT"

    # Build test arguments
    local test_args=("-v")

    if [[ -n "$SPECIFIC_TEST" ]]; then
        test_args+=("-run" "Test${SPECIFIC_TEST}Resource")
        info "Running specific test: Test${SPECIFIC_TEST}Resource"
    elif [[ "$RUN_MODE" == "all" ]]; then
        export RUN_ALL_TESTS=1
        test_args+=("-run" "TestAllResources")
        info "Running all tests in sequence"
    fi

    # Set skip flags for quick mode
    if [[ "$RUN_MODE" == "quick" ]]; then
        export SKIP_INSTANCE_TEST=1
        export SKIP_CONTAINER_TEST=1
        export SKIP_SERVERLESS_JOB_TEST=1
        warn "Quick mode: skipping instance, container, and serverless_job tests"
    fi

    # Set TF_CMD for the tests
    export TF_CMD="$TF_TOOL"

    # Run the tests
    echo ""
    info "Test output:"
    echo "─────────────────────────────────────────────────────────────"

    if go test ./test/integration/... "${test_args[@]}"; then
        echo "─────────────────────────────────────────────────────────────"
        echo ""
        success "All integration tests passed!"
    else
        echo "─────────────────────────────────────────────────────────────"
        echo ""
        error "Some tests failed"
        exit 1
    fi
}

# Clean up temporary test directories and resources
cleanup_test_resources() {
    info "Cleaning up test resources..."

    local found=0
    local cleanup_failed=0

    # Find all temp test directories
    for dir in /tmp/verda-integration-test-* /tmp/verda-test-*; do
        if [[ -d "$dir" ]]; then
            found=1
            info "Found test directory: $dir"

            # Try to run terraform destroy if state exists
            if [[ -f "$dir/terraform.tfstate" ]]; then
                info "Running $TF_TOOL destroy in $dir..."
                cd "$dir"
                if $TF_TOOL destroy -auto-approve 2>/dev/null; then
                    success "Resources destroyed in $dir"
                else
                    warn "Failed to destroy resources in $dir (may already be deleted)"
                    cleanup_failed=1
                fi
                cd - > /dev/null
            fi

            # Remove the directory
            rm -rf "$dir"
            success "Removed: $dir"
        fi
    done

    if [[ $found -eq 0 ]]; then
        info "No temporary test directories found"
    fi

    return $cleanup_failed
}

# Print summary
print_summary() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "                    TEST SUMMARY"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    info "CLI Tool: $TF_TOOL"
    info "Mode: $RUN_MODE"
    [[ -n "$SPECIFIC_TEST" ]] && info "Specific Test: $SPECIFIC_TEST"
    echo ""
}

# Main
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "     Verda Terraform Provider - Integration Tests"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    load_env
    check_env
    check_tf_tool

    # If cleanup only, just run cleanup and exit
    if [[ "$CLEANUP_ONLY" == true ]]; then
        cleanup_test_resources
        success "Cleanup complete!"
        exit 0
    fi

    build_provider
    install_provider_locally

    # Run tests and capture result
    local test_result=0
    run_tests || test_result=$?

    # Always run cleanup after tests
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "                    CLEANUP"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    cleanup_test_resources || true

    print_summary

    # Exit with test result
    exit $test_result
}

main "$@"
