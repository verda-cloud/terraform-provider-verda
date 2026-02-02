# Integration Tests

These integration tests verify that the documentation examples work correctly with both Terraform and OpenTofu.

## Purpose

The tests act as a "user" following the documentation exactly:
1. Copy the test `.tf` files (based on documentation examples) to a temp directory
2. Run `terraform init` (or `tofu init`)
3. Run `terraform apply -auto-approve` (or `tofu apply`)
4. Verify resources are created successfully
5. Run `terraform destroy -auto-approve` (cleanup)

## Quick Start (Recommended)

Use the automated test script:

```bash
# Setup credentials
export VERDA_CLIENT_ID="your-client-id"
export VERDA_CLIENT_SECRET="your-client-secret"

# Run tests with Terraform
./scripts/run-integration-tests.sh --terraform

# Run tests with OpenTofu
./scripts/run-integration-tests.sh --opentofu

# Run quick tests only (skip expensive GPU resources)
./scripts/run-integration-tests.sh --quick

# Run all tests including GPU instances
./scripts/run-integration-tests.sh --all

# Run a specific test
./scripts/run-integration-tests.sh --test SSHKey
```

The script automatically:
- Loads `.env` file if present
- Builds the provider
- Installs it locally for testing
- Runs the tests
- Provides colored output and summary

## Manual Setup

### Prerequisites

1. **API Credentials**: Set your Verda API credentials as environment variables:
   ```bash
   export VERDA_CLIENT_ID="your-client-id"
   export VERDA_CLIENT_SECRET="your-client-secret"
   ```

   Or create a `.env` file (copy from `.env.example`):
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

2. **Terraform or OpenTofu**: Install either:
   - [Terraform](https://www.terraform.io/downloads) (v1.0+)
   - [OpenTofu](https://opentofu.org/docs/intro/install/) (v1.6+)

3. **Provider Binary**: Build and install the provider locally:
   ```bash
   # From the repository root
   go build -o terraform-provider-verda

   # Install to local plugins directory (use version 99.0.0 for dev)
   mkdir -p ~/.terraform.d/plugins/verda-cloud/verda/99.0.0/$(go env GOOS)_$(go env GOARCH)
   cp terraform-provider-verda ~/.terraform.d/plugins/verda-cloud/verda/99.0.0/$(go env GOOS)_$(go env GOARCH)/
   ```

## Running Tests Manually

### Run all tests with Terraform

```bash
VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy go test ./test/integration/... -v
```

### Run all tests with OpenTofu

```bash
VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy TF_CMD=tofu go test ./test/integration/... -v
```

### Run a specific test

```bash
# Test only SSH key resource
VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy go test ./test/integration/... -v -run TestSSHKeyResource

# Test only volume resource
VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy go test ./test/integration/... -v -run TestVolumeResource
```

### Skip expensive tests

Some tests create real GPU instances or containers which may incur costs. You can skip them:

```bash
# Skip instance and container tests
VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy \
  SKIP_INSTANCE_TEST=1 \
  SKIP_CONTAINER_TEST=1 \
  SKIP_SERVERLESS_JOB_TEST=1 \
  go test ./test/integration/... -v
```

### Run all tests in sequence (for CI/CD)

```bash
VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy RUN_ALL_TESTS=1 go test ./test/integration/... -v -run TestAllResources
```

## Test Files

Each test uses Terraform configurations that mirror the documentation examples:

| Test | Resource | Test File |
|------|----------|-----------|
| `TestSSHKeyResource` | `verda_ssh_key` | `testdata/ssh_key/main.tf` |
| `TestStartupScriptResource` | `verda_startup_script` | `testdata/startup_script/main.tf` |
| `TestVolumeResource` | `verda_volume` | `testdata/volume/main.tf` |
| `TestContainerRegistryCredentialsResource` | `verda_container_registry_credentials` | `testdata/container_registry_credentials/main.tf` |
| `TestInstanceResource` | `verda_instance` | `testdata/instance/main.tf` |
| `TestContainerResource` | `verda_container` | `testdata/container/main.tf` |
| `TestServerlessJobResource` | `verda_serverless_job` | `testdata/serverless_job/main.tf` |

## Configurable Test Variables

Some tests use Terraform variables that can be overridden via environment variables. This allows testing against different environments (staging, production) without modifying test files or documentation.

### Available Variables

| Variable | Environment Variable | Default | Description |
|----------|---------------------|---------|-------------|
| `instance_type` | `TF_VAR_instance_type` | `1B200.30V` | GPU instance type |
| `instance_image` | `TF_VAR_instance_image` | `ubuntu-24.04-cuda-12.8-open-docker` | Instance image |
| `instance_location` | `TF_VAR_instance_location` | `FIN-03` | Instance location |

### Example: Testing with Staging Instance Type

```bash
# Set staging instance type (RTX PRO 6000)
export TF_VAR_instance_type="1RTXPRO6000.30V"

# Or add to .env file
echo 'TF_VAR_instance_type=1RTXPRO6000.30V' >> .env

# Run instance test
./scripts/run-integration-tests.sh --test Instance
```

### How It Works

1. Variables are defined in `testdata/variables.tf` with defaults matching documentation
2. Terraform automatically reads `TF_VAR_*` environment variables
3. Test files use `var.instance_type` instead of hardcoded values
4. No need to modify docs or test files for different environments

## Adding New Tests

1. Create a new directory under `testdata/` with the resource name
2. Add a `main.tf` file that follows the documentation example exactly
3. Add a new test function in `integration_test.go` following the existing pattern

## Troubleshooting

### Tests skip with "VERDA_CLIENT_ID not set"

Ensure you've exported the environment variables in your current shell:
```bash
export VERDA_CLIENT_ID="your-client-id"
export VERDA_CLIENT_SECRET="your-client-secret"
```

### "terraform: command not found"

Install Terraform or OpenTofu, or set the `TF_CMD` environment variable to point to your executable:
```bash
export TF_CMD=/path/to/terraform
```

### Provider not found

Make sure you've built and installed the provider locally. See Prerequisites above.

### Resources not cleaning up

If a test fails mid-way, resources might not be destroyed. You can manually clean up:
```bash
cd /tmp/verda-integration-test-*
terraform destroy -auto-approve
```
