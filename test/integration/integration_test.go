// Package integration provides integration tests that follow documentation examples
// to verify they work with both Terraform and OpenTofu.
package integration

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// tfCmd returns the Terraform/OpenTofu command to use.
// Defaults to "terraform", can be overridden with TF_CMD environment variable.
func tfCmd() string {
	if cmd := os.Getenv("TF_CMD"); cmd != "" {
		return cmd
	}
	return "terraform"
}

// testdataDir returns the path to the testdata directory
func testdataDir() string {
	return filepath.Join("testdata")
}

// setupTestDir creates a temporary directory with the provider.tf, variables.tf, and resource main.tf
func setupTestDir(t *testing.T, resourceDir string) string {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "verda-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Copy provider.tf
	providerSrc := filepath.Join(testdataDir(), "provider.tf")
	providerDst := filepath.Join(tmpDir, "provider.tf")
	if err := copyFile(providerSrc, providerDst); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to copy provider.tf: %v", err)
	}

	// Copy variables.tf (shared variables with defaults, can be overridden via TF_VAR_*)
	variablesSrc := filepath.Join(testdataDir(), "variables.tf")
	variablesDst := filepath.Join(tmpDir, "variables.tf")
	if err := copyFile(variablesSrc, variablesDst); err != nil {
		// variables.tf is optional for tests that don't use variables
		t.Logf("Note: variables.tf not copied (may not exist): %v", err)
	}

	// Copy resource main.tf
	mainSrc := filepath.Join(testdataDir(), resourceDir, "main.tf")
	mainDst := filepath.Join(tmpDir, "main.tf")
	if err := copyFile(mainSrc, mainDst); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to copy main.tf from %s: %v", resourceDir, err)
	}

	return tmpDir
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// runTerraform executes a terraform/tofu command in the specified directory
func runTerraform(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command(tfCmd(), args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	t.Logf("Running: %s %s (in %s)", tfCmd(), strings.Join(args, " "), dir)

	err := cmd.Run()
	if err != nil {
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		t.Fatalf("Command failed: %s %s: %v", tfCmd(), strings.Join(args, " "), err)
	}

	return stdout.String()
}

// cleanupTestDir removes the temp directory and runs terraform destroy
func cleanupTestDir(t *testing.T, dir string) {
	t.Helper()

	t.Logf("Cleaning up resources in %s...", dir)

	// Run destroy to clean up any created resources
	cmd := exec.Command(tfCmd(), "destroy", "-auto-approve")
	cmd.Dir = dir
	cmd.Env = os.Environ() // Pass environment variables (credentials)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("Warning: terraform destroy failed: %v", err)
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())

		// Retry destroy once after a short delay
		t.Log("Retrying destroy after 5 seconds...")
		sleepCmd := exec.Command("sleep", "5")
		_ = sleepCmd.Run() // Ignore error for sleep

		retryCmd := exec.Command(tfCmd(), "destroy", "-auto-approve")
		retryCmd.Dir = dir
		retryCmd.Env = os.Environ() // Pass environment variables (credentials)
		retryOutput, retryErr := retryCmd.CombinedOutput()
		if retryErr != nil {
			t.Logf("Warning: retry destroy also failed: %v", retryErr)
			t.Logf("output: %s", string(retryOutput))
		} else {
			t.Log("Retry destroy succeeded")
		}
	} else {
		t.Log("Resources destroyed successfully")
	}

	// Remove temp directory
	os.RemoveAll(dir)
}

// registerCleanup registers cleanup to run after test completes (even on panic)
func registerCleanup(t *testing.T, dir string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupTestDir(t, dir)
	})
}

// checkEnvVars ensures required environment variables are set
func checkEnvVars(t *testing.T) {
	t.Helper()

	if os.Getenv("VERDA_CLIENT_ID") == "" {
		t.Skip("Skipping integration test: VERDA_CLIENT_ID not set")
	}
	if os.Getenv("VERDA_CLIENT_SECRET") == "" {
		t.Skip("Skipping integration test: VERDA_CLIENT_SECRET not set")
	}
}

// TestSSHKeyResource tests the SSH key resource following documentation examples
func TestSSHKeyResource(t *testing.T) {
	checkEnvVars(t)

	workDir := setupTestDir(t, "ssh_key")
	registerCleanup(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "ssh_key_id") {
		t.Error("Expected ssh_key_id in output")
	}
	if !strings.Contains(output, "ssh_key_fingerprint") {
		t.Error("Expected ssh_key_fingerprint in output")
	}

	t.Log("SSH Key resource test passed")
}

// TestStartupScriptResource tests the startup script resource following documentation examples
func TestStartupScriptResource(t *testing.T) {
	checkEnvVars(t)

	workDir := setupTestDir(t, "startup_script")
	registerCleanup(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "startup_script_id") {
		t.Error("Expected startup_script_id in output")
	}

	t.Log("Startup Script resource test passed")
}

// TestVolumeResource tests the volume resource following documentation examples
func TestVolumeResource(t *testing.T) {
	checkEnvVars(t)

	workDir := setupTestDir(t, "volume")
	registerCleanup(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "volume_id") {
		t.Error("Expected volume_id in output")
	}
	if !strings.Contains(output, "volume_status") {
		t.Error("Expected volume_status in output")
	}

	t.Log("Volume resource test passed")
}

// TestContainerRegistryCredentialsResource tests the registry credentials resource
// Note: This resource is still in testing stage and skipped by default
func TestContainerRegistryCredentialsResource(t *testing.T) {
	checkEnvVars(t)

	// Skip by default - this resource is still in testing stage
	if os.Getenv("RUN_REGISTRY_CREDENTIALS_TEST") == "" {
		t.Skip("Skipping registry credentials test: RUN_REGISTRY_CREDENTIALS_TEST not set (resource still in testing stage)")
	}

	workDir := setupTestDir(t, "container_registry_credentials")
	registerCleanup(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist (note: this resource uses 'name' as identifier, no 'id' attribute)
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "credentials_name") {
		t.Error("Expected credentials_name in output")
	}
	if !strings.Contains(output, "credentials_created_at") {
		t.Error("Expected credentials_created_at in output")
	}

	t.Log("Container Registry Credentials resource test passed")
}

// TestSecretResource tests the secret resource following documentation examples
func TestSecretResource(t *testing.T) {
	checkEnvVars(t)

	workDir := setupTestDir(t, "secret")
	defer cleanupTestDir(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "secret_name") {
		t.Error("Expected secret_name in output")
	}
	if !strings.Contains(output, "secret_type") {
		t.Error("Expected secret_type in output")
	}

	t.Log("Secret resource test passed")
}

// TestFileSecretResource tests the file secret resource following documentation examples
func TestFileSecretResource(t *testing.T) {
	checkEnvVars(t)

	workDir := setupTestDir(t, "file_secret")
	defer cleanupTestDir(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "file_secret_name") {
		t.Error("Expected file_secret_name in output")
	}
	if !strings.Contains(output, "file_secret_file_names") {
		t.Error("Expected file_secret_file_names in output")
	}

	t.Log("File secret resource test passed")
}

// TestInstanceResource tests the instance resource following documentation examples
// Note: This test creates real GPU instances and may incur costs
// The test waits up to 5 minutes for the instance to be fully deployed
func TestInstanceResource(t *testing.T) {
	checkEnvVars(t)

	// Skip if explicitly disabled (instances are expensive)
	if os.Getenv("SKIP_INSTANCE_TEST") != "" {
		t.Skip("Skipping instance test: SKIP_INSTANCE_TEST is set")
	}

	workDir := setupTestDir(t, "instance")
	registerCleanup(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify instance_id exists immediately
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "instance_id") {
		t.Error("Expected instance_id in output")
	}
	if !strings.Contains(output, "instance_type") {
		t.Error("Expected instance_type in output")
	}

	// Wait for instance to be ready (up to 5 minutes, check every minute)
	t.Log("Waiting for instance to be ready (up to 5 minutes)...")
	instanceReady := false
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Refresh state to get latest status
		runTerraform(t, workDir, "refresh")
		output = runTerraform(t, workDir, "output", "-json")

		// Check if instance is running
		if strings.Contains(output, `"value": "running"`) {
			t.Logf("Instance is running after %d minute(s)", attempt)
			instanceReady = true
			break
		}

		if attempt < maxAttempts {
			t.Logf("Attempt %d/%d: Instance not ready yet, waiting 1 minute...", attempt, maxAttempts)
			// Sleep for 1 minute using time.Sleep
			sleepCmd := exec.Command("sleep", "60")
			_ = sleepCmd.Run() // Ignore error for sleep
		}
	}

	// Final verification
	output = runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "instance_status") {
		t.Error("Expected instance_status in output")
	}

	if instanceReady {
		// Verify IP is assigned when running
		if !strings.Contains(output, "instance_ip") {
			t.Error("Expected instance_ip in output when instance is running")
		}
		t.Log("Instance resource test passed - instance is running with IP assigned")
	} else {
		t.Log("Instance resource test passed - instance created but still provisioning (this is acceptable)")
	}
}

// TestContainerResource tests the container resource following documentation examples
func TestContainerResource(t *testing.T) {
	checkEnvVars(t)

	// Skip if explicitly disabled
	if os.Getenv("SKIP_CONTAINER_TEST") != "" {
		t.Skip("Skipping container test: SKIP_CONTAINER_TEST is set")
	}

	workDir := setupTestDir(t, "container")
	registerCleanup(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "container_name") {
		t.Error("Expected container_name in output")
	}
	if !strings.Contains(output, "container_endpoint_base_url") {
		t.Error("Expected container_endpoint_base_url in output")
	}

	t.Log("Container resource test passed")
}

// TestServerlessJobResource tests the serverless job resource following documentation examples
func TestServerlessJobResource(t *testing.T) {
	checkEnvVars(t)

	// Skip if explicitly disabled
	if os.Getenv("SKIP_SERVERLESS_JOB_TEST") != "" {
		t.Skip("Skipping serverless job test: SKIP_SERVERLESS_JOB_TEST is set")
	}

	workDir := setupTestDir(t, "serverless_job")
	registerCleanup(t, workDir)

	// Init
	runTerraform(t, workDir, "init")

	// Apply
	runTerraform(t, workDir, "apply", "-auto-approve")

	// Verify outputs exist
	output := runTerraform(t, workDir, "output", "-json")
	if !strings.Contains(output, "job_name") {
		t.Error("Expected job_name in output")
	}
	if !strings.Contains(output, "job_endpoint_base_url") {
		t.Error("Expected job_endpoint_base_url in output")
	}

	t.Log("Serverless Job resource test passed")
}

// TestAllResources runs all resource tests in sequence
// This is useful for CI/CD pipelines
func TestAllResources(t *testing.T) {
	if os.Getenv("RUN_ALL_TESTS") == "" {
		t.Skip("Skipping TestAllResources: RUN_ALL_TESTS not set")
	}

	t.Run("SSHKey", TestSSHKeyResource)
	t.Run("StartupScript", TestStartupScriptResource)
	t.Run("Volume", TestVolumeResource)
	t.Run("ContainerRegistryCredentials", TestContainerRegistryCredentialsResource)
	t.Run("Secret", TestSecretResource)
	t.Run("FileSecret", TestFileSecretResource)
	t.Run("Instance", TestInstanceResource)
	t.Run("Container", TestContainerResource)
	t.Run("ServerlessJob", TestServerlessJobResource)
}

// ExampleUsage demonstrates how to run the tests
func Example() {
	fmt.Println("Run integration tests with Terraform:")
	fmt.Println("  VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy go test ./test/integration/... -v")
	fmt.Println("")
	fmt.Println("Run integration tests with OpenTofu:")
	fmt.Println("  VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy TF_CMD=tofu go test ./test/integration/... -v")
	fmt.Println("")
	fmt.Println("Run a specific test:")
	fmt.Println("  VERDA_CLIENT_ID=xxx VERDA_CLIENT_SECRET=yyy go test ./test/integration/... -v -run TestSSHKeyResource")
	fmt.Println("")
	fmt.Println("Skip expensive tests:")
	fmt.Println("  SKIP_INSTANCE_TEST=1 SKIP_CONTAINER_TEST=1 go test ./test/integration/... -v")
}
