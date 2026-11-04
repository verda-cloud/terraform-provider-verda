// Copyright 2026 Verda Cloud Oy
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// planExitCode runs "plan -detailed-exitcode" and returns 0 (no changes) or
// 2 (changes) with the plan output; any other result fails the test.
func planExitCode(t *testing.T, dir string, args ...string) (int, string) {
	t.Helper()

	args = append([]string{"plan", "-detailed-exitcode", "-no-color"}, args...)
	cmd := exec.Command(tfCmd(), args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	t.Logf("Running: %s %s (in %s)", tfCmd(), strings.Join(args, " "), dir)

	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, stdout.String()
	case errors.As(err, &exitErr) && exitErr.ExitCode() == 2:
		return 2, stdout.String()
	}
	t.Logf("stdout: %s", stdout.String())
	t.Logf("stderr: %s", stderr.String())
	t.Fatalf("plan failed: %v", err)
	return 0, ""
}

// outputs returns the root module outputs as strings (null as "").
func outputs(t *testing.T, dir string) map[string]string {
	t.Helper()

	var raw map[string]struct {
		Value any `json:"value"`
	}
	if err := json.Unmarshal([]byte(runTerraform(t, dir, "output", "-json")), &raw); err != nil {
		t.Fatalf("parse outputs: %v", err)
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		if v.Value != nil {
			out[k] = fmt.Sprint(v.Value)
		}
	}
	return out
}

// requireNoChanges fails the test if a plan with args is not empty.
func requireNoChanges(t *testing.T, dir, why string, args ...string) {
	t.Helper()
	if code, plan := planExitCode(t, dir, args...); code != 0 {
		t.Fatalf("%s: expected an empty plan, got:\n%s", why, plan)
	}
}

// TestPrivateNetworkResource follows the private network docs end to end: private network, subnet, private network
// instances, a route, data sources, in-place updates, import, and destroy
// while instances are still being deleted (the 409 retry).
//
// It creates three instances, so it is skipped with SKIP_INSTANCE_TEST, or on
// its own with SKIP_PRIVATE_NETWORK_TEST. Needs a location where private
// networks are available (TF_VAR_instance_location, default FIN-03).
//
//	go test ./test/integration/ -run TestPrivateNetworkResource -v -timeout 60m
//
// VERDA_TEST_DEFAULT_PRIVATE_NETWORK=1 also tests verda_private_network_default and
// verda_subnet_default. That changes the project's default private network and
// subnet, so only set it in a dedicated test project.
func TestPrivateNetworkResource(t *testing.T) {
	checkEnvVars(t)
	if os.Getenv("SKIP_PRIVATE_NETWORK_TEST") != "" {
		t.Skip("Skipping private network test: SKIP_PRIVATE_NETWORK_TEST is set")
	}
	if os.Getenv("SKIP_INSTANCE_TEST") != "" {
		t.Skip("Skipping private network test: it creates instances and SKIP_INSTANCE_TEST is set")
	}

	// Variables go through the environment so that the cleanup's plain
	// "destroy" sees them too.
	t.Setenv("TF_VAR_name_suffix", fmt.Sprintf("%d", time.Now().Unix()))
	t.Setenv("TF_VAR_manage_defaults", fmt.Sprint(os.Getenv("VERDA_TEST_DEFAULT_PRIVATE_NETWORK") != ""))

	workDir := setupTestDir(t, "private_network")
	registerCleanup(t, workDir)

	runTerraform(t, workDir, "init")

	// 1. Create everything; the plan afterwards must be empty.
	runTerraform(t, workDir, "apply", "-auto-approve")
	out := outputs(t, workDir)
	for _, k := range []string{"private_network_id", "subnet_id", "route_id", "nat_a_id", "worker_id", "worker_private_ip"} {
		if out[k] == "" {
			t.Fatalf("output %s is empty; outputs: %v", k, out)
		}
	}
	if !strings.HasPrefix(out["private_network_id"], "pn_") || !strings.HasPrefix(out["subnet_id"], "sn_") || !strings.HasPrefix(out["route_id"], "rt_") {
		t.Errorf("unexpected id formats: %v", out)
	}
	if out["worker_private_network_id"] != out["private_network_id"] {
		t.Errorf("worker private_network_id = %q, want %q", out["worker_private_network_id"], out["private_network_id"])
	}
	if out["worker_private_ip"] != "10.231.0.50" {
		t.Errorf("worker private_ip_address = %q, want 10.231.0.50", out["worker_private_ip"])
	}
	if out["worker_public_ip"] != "" {
		t.Errorf("worker with public_ip = none has ip %q", out["worker_public_ip"])
	}
	if out["ds_private_network_id"] != out["private_network_id"] {
		t.Errorf("data.verda_private_network id = %q, want %q", out["ds_private_network_id"], out["private_network_id"])
	}
	if out["ds_subnet_default_route_instance_id"] != out["nat_a_id"] {
		t.Errorf("data.verda_subnet default_route_instance_id = %q, want %q", out["ds_subnet_default_route_instance_id"], out["nat_a_id"])
	}
	requireNoChanges(t, workDir, "after create")

	// 2. Renaming the private network and subnet updates them in place.
	rename := []string{"-var", "private_network_name=itest-pn-renamed", "-var", "subnet_name=itest-app-renamed"}
	code, plan := planExitCode(t, workDir, rename...)
	if code != 2 || !strings.Contains(plan, "0 to add, 2 to change, 0 to destroy") {
		t.Fatalf("rename: expected 2 in-place changes, got:\n%s", plan)
	}
	runTerraform(t, workDir, append([]string{"apply", "-auto-approve"}, rename...)...)
	requireNoChanges(t, workDir, "after rename", rename...)

	// 3. Moving the route to the other NAT instance updates it in place.
	move := append(rename, "-var", "route_via=b")
	code, plan = planExitCode(t, workDir, move...)
	if code != 2 || !strings.Contains(plan, "0 to add, 1 to change, 0 to destroy") {
		t.Fatalf("route move: expected 1 in-place change, got:\n%s", plan)
	}
	runTerraform(t, workDir, append([]string{"apply", "-auto-approve"}, move...)...)
	requireNoChanges(t, workDir, "after route move", move...)
	if moved := outputs(t, workDir); moved["route_id"] != out["route_id"] {
		t.Logf("note: the route id changed when the route moved (%s -> %s)", out["route_id"], moved["route_id"])
	}

	// 4. Forget and re-import every resource; the plan must still be empty.
	out = outputs(t, workDir)
	imports := map[string]string{
		"verda_private_network.test": out["private_network_id"],
		"verda_subnet.app":           out["private_network_id"] + "/" + out["subnet_id"],
		"verda_network_route.egress": out["private_network_id"] + "/" + out["subnet_id"] + "/" + out["route_id"],
		"verda_instance.worker":      out["worker_id"],
		`verda_instance.nat["a"]`:    out["nat_a_id"],
		`verda_instance.nat["b"]`:    out["nat_b_id"],
	}
	if os.Getenv("VERDA_TEST_DEFAULT_PRIVATE_NETWORK") != "" {
		imports["verda_private_network_default.this[0]"] = "default"
		imports["verda_subnet_default.this[0]"] = out["private_network_id"] + "/" + os.Getenv("TF_VAR_instance_location")
		if os.Getenv("TF_VAR_instance_location") == "" {
			imports["verda_subnet_default.this[0]"] = out["private_network_id"] + "/FIN-03"
		}
	}
	for addr, id := range imports {
		runTerraform(t, workDir, "state", "rm", addr)
		runTerraform(t, workDir, append([]string{"import"}, append(move, addr, id)...)...)
	}
	// Import records the worker's private_ip as "auto"; the config pins the
	// address it has, which is the same placement. That may update state in
	// place, but must never replace anything.
	if code, plan := planExitCode(t, workDir, move...); code == 2 {
		if !strings.Contains(plan, "0 to add,") || !strings.Contains(plan, "0 to destroy") {
			t.Fatalf("after import: expected no replacement, got:\n%s", plan)
		}
		runTerraform(t, workDir, append([]string{"apply", "-auto-approve"}, move...)...)
	}
	requireNoChanges(t, workDir, "after import", move...)

	// 5. Cleanup destroys everything. Instance deletion is asynchronous, so
	// the subnet and private network deletes exercise the 409 retry.
	t.Log("Private network resource test passed")
}
