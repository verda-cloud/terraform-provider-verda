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

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func shortenManagedKubernetesPoll(t *testing.T) {
	t.Helper()
	managedKubernetesPollInterval = time.Millisecond
	t.Cleanup(func() { managedKubernetesPollInterval = 10 * time.Second })
}

func withManagedKubernetesTimeout(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func TestManagedKubernetesClusterIDRoundTrip(t *testing.T) {
	location, name, err := managedKubernetesParseClusterID(managedKubernetesFormatClusterID("FIN-03", "my-cluster"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if location != "FIN-03" || name != "my-cluster" {
		t.Fatalf("got %q/%q", location, name)
	}
}

func TestManagedKubernetesParseClusterIDInvalid(t *testing.T) {
	for _, id := range []string{"", "FIN-03", "FIN-03/", "/my-cluster", "a/b/c"} {
		if _, _, err := managedKubernetesParseClusterID(id); err == nil {
			t.Errorf("expected error for %q", id)
		}
	}
}

func TestManagedKubernetesNodePoolIDRoundTrip(t *testing.T) {
	location, cluster, pool, err := managedKubernetesParseNodePoolID(managedKubernetesFormatNodePoolID("FIN-03", "my-cluster", "gpu"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if location != "FIN-03" || cluster != "my-cluster" || pool != "gpu" {
		t.Fatalf("got %q/%q/%q", location, cluster, pool)
	}
}

func TestManagedKubernetesParseNodePoolIDInvalid(t *testing.T) {
	for _, id := range []string{"", "FIN-03", "a/b", "a/b/c/d", "a//c"} {
		if _, _, _, err := managedKubernetesParseNodePoolID(id); err == nil {
			t.Errorf("expected error for %q", id)
		}
	}
}

func TestManagedKubernetesParseKubeconfigImportID(t *testing.T) {
	location, cluster, id, err := managedKubernetesParseKubeconfigImportID("FIN-03/my-cluster/kc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if location != "FIN-03" || cluster != "my-cluster" || id != "kc-123" {
		t.Fatalf("got %q/%q/%q", location, cluster, id)
	}
	for _, bad := range []string{"kc-123", "FIN-03/kc-123", "a/b/c/d"} {
		if _, _, _, err := managedKubernetesParseKubeconfigImportID(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestManagedKubernetesIsNotFound(t *testing.T) {
	tests := map[string]struct {
		err  error
		want bool
	}{
		"404":             {err: &verda.APIError{StatusCode: 404}, want: true},
		"wrapped 404":     {err: fmt.Errorf("get: %w", &verda.APIError{StatusCode: 404}), want: true},
		"500":             {err: &verda.APIError{StatusCode: 500}, want: false},
		"plain error":     {err: errors.New("boom"), want: false},
		"404-shaped text": {err: errors.New("API error 404: not found"), want: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := managedKubernetesIsNotFound(tc.err); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

type fakeManagedKubernetesClusterGetter struct {
	phases []string
	calls  int
	// goneAfter: return 404 once phases are exhausted; otherwise keep
	// returning the last phase.
	goneAfter bool
}

func (f *fakeManagedKubernetesClusterGetter) Get(ctx context.Context, location, clusterName string) (*verda.ManagedKubernetesCluster, error) {
	if f.calls < len(f.phases) {
		phase := f.phases[f.calls]
		f.calls++
		return &verda.ManagedKubernetesCluster{Phase: phase}, nil
	}
	if f.goneAfter {
		return nil, &verda.APIError{StatusCode: 404}
	}
	return &verda.ManagedKubernetesCluster{Phase: f.phases[len(f.phases)-1]}, nil
}

func TestManagedKubernetesWaitForClusterRunning(t *testing.T) {
	shortenManagedKubernetesPoll(t)

	t.Run("transitions to running", func(t *testing.T) {
		f := &fakeManagedKubernetesClusterGetter{phases: []string{"provisioning", "provisioning", "running"}}
		cluster, err := managedKubernetesWaitForClusterRunning(withManagedKubernetesTimeout(t, time.Second), f, "FIN-03", "c1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cluster.Phase != "running" {
			t.Fatalf("got phase %q", cluster.Phase)
		}
	})

	t.Run("times out with last phase", func(t *testing.T) {
		f := &fakeManagedKubernetesClusterGetter{phases: []string{"provisioning"}}
		_, err := managedKubernetesWaitForClusterRunning(withManagedKubernetesTimeout(t, 20*time.Millisecond), f, "FIN-03", "c1")
		if err == nil || !strings.Contains(err.Error(), `last phase "provisioning"`) {
			t.Fatalf("expected timeout with last phase, got: %v", err)
		}
	})

	t.Run("polls through degraded", func(t *testing.T) {
		f := &fakeManagedKubernetesClusterGetter{phases: []string{"degraded", "running"}}
		if _, err := managedKubernetesWaitForClusterRunning(withManagedKubernetesTimeout(t, time.Second), f, "FIN-03", "c1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestManagedKubernetesWaitForClusterDeleted(t *testing.T) {
	shortenManagedKubernetesPoll(t)

	t.Run("gone after deleting", func(t *testing.T) {
		f := &fakeManagedKubernetesClusterGetter{phases: []string{"deleting", "deleting"}, goneAfter: true}
		if err := managedKubernetesWaitForClusterDeleted(withManagedKubernetesTimeout(t, time.Second), f, "FIN-03", "c1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("times out while still present", func(t *testing.T) {
		f := &fakeManagedKubernetesClusterGetter{phases: []string{"deleting"}}
		err := managedKubernetesWaitForClusterDeleted(withManagedKubernetesTimeout(t, 20*time.Millisecond), f, "FIN-03", "c1")
		if err == nil || !strings.Contains(err.Error(), "still exists") {
			t.Fatalf("expected delete timeout, got: %v", err)
		}
	})
}

type fakeManagedKubernetesNodePoolGetter struct {
	pools     []*verda.ManagedKubernetesNodePool
	calls     int
	goneAfter bool
}

func (f *fakeManagedKubernetesNodePoolGetter) GetNodePool(ctx context.Context, location, clusterName, poolName string) (*verda.ManagedKubernetesNodePool, error) {
	if f.calls < len(f.pools) {
		p := f.pools[f.calls]
		f.calls++
		return p, nil
	}
	if f.goneAfter {
		return nil, &verda.APIError{StatusCode: 404}
	}
	return f.pools[len(f.pools)-1], nil
}

func TestManagedKubernetesWaitForNodePoolActive(t *testing.T) {
	shortenManagedKubernetesPoll(t)

	t.Run("transitions to active", func(t *testing.T) {
		f := &fakeManagedKubernetesNodePoolGetter{pools: []*verda.ManagedKubernetesNodePool{
			{State: "creating"}, {State: "active"},
		}}
		pool, err := managedKubernetesWaitForNodePoolActive(withManagedKubernetesTimeout(t, time.Second), f, "FIN-03", "c1", "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pool.State != "active" {
			t.Fatalf("got state %q", pool.State)
		}
	})

	t.Run("timeout surfaces pool errors", func(t *testing.T) {
		f := &fakeManagedKubernetesNodePoolGetter{pools: []*verda.ManagedKubernetesNodePool{{
			State: "degraded",
			Errors: []verda.ManagedKubernetesNodePoolError{
				{Reason: "insufficient_funds", Message: "not enough credit"},
			},
		}}}
		_, err := managedKubernetesWaitForNodePoolActive(withManagedKubernetesTimeout(t, 20*time.Millisecond), f, "FIN-03", "c1", "p1")
		if err == nil || !strings.Contains(err.Error(), "insufficient_funds: not enough credit") {
			t.Fatalf("expected timeout with pool errors, got: %v", err)
		}
	})
}

func TestManagedKubernetesWaitForNodePoolDeleted(t *testing.T) {
	shortenManagedKubernetesPoll(t)
	f := &fakeManagedKubernetesNodePoolGetter{
		pools:     []*verda.ManagedKubernetesNodePool{{State: "deleting"}},
		goneAfter: true,
	}
	if err := managedKubernetesWaitForNodePoolDeleted(withManagedKubernetesTimeout(t, time.Second), f, "FIN-03", "c1", "p1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeManagedKubernetesKubeconfigGetter struct {
	statuses  []string
	calls     int
	goneAfter bool
}

func (f *fakeManagedKubernetesKubeconfigGetter) GetKubeconfig(ctx context.Context, location, clusterName, id string) (*verda.ManagedKubernetesKubeconfig, error) {
	if f.calls < len(f.statuses) {
		status := f.statuses[f.calls]
		f.calls++
		return &verda.ManagedKubernetesKubeconfig{Status: status}, nil
	}
	if f.goneAfter {
		return nil, &verda.APIError{StatusCode: 404}
	}
	return &verda.ManagedKubernetesKubeconfig{Status: f.statuses[len(f.statuses)-1]}, nil
}

func TestManagedKubernetesWaitForKubeconfigReady(t *testing.T) {
	shortenManagedKubernetesPoll(t)

	t.Run("pending to ready", func(t *testing.T) {
		f := &fakeManagedKubernetesKubeconfigGetter{statuses: []string{"pending", "ready"}}
		kc, err := managedKubernetesWaitForKubeconfigReady(withManagedKubernetesTimeout(t, time.Second), f, "FIN-03", "c1", "kc-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if kc.Status != "ready" {
			t.Fatalf("got status %q", kc.Status)
		}
	})

	t.Run("stuck pending times out", func(t *testing.T) {
		f := &fakeManagedKubernetesKubeconfigGetter{statuses: []string{"pending"}}
		_, err := managedKubernetesWaitForKubeconfigReady(withManagedKubernetesTimeout(t, 20*time.Millisecond), f, "FIN-03", "c1", "kc-1")
		if err == nil || !strings.Contains(err.Error(), `last status "pending"`) {
			t.Fatalf("expected timeout with last status, got: %v", err)
		}
	})
}

func TestManagedKubernetesWaitForKubeconfigDeleted(t *testing.T) {
	shortenManagedKubernetesPoll(t)
	f := &fakeManagedKubernetesKubeconfigGetter{statuses: []string{"ready"}, goneAfter: true}
	if err := managedKubernetesWaitForKubeconfigDeleted(withManagedKubernetesTimeout(t, time.Second), f, "FIN-03", "c1", "kc-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func validateManagedKubernetesString(v validator.String, value types.String) bool {
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: value,
	}, resp)
	return resp.Diagnostics.HasError()
}

func TestManagedKubernetesDNS1123LabelValidator(t *testing.T) {
	v := managedKubernetesDNS1123LabelValidator{maxLen: 30}
	tests := map[string]struct {
		value   types.String
		wantErr bool
	}{
		"simple":        {types.StringValue("gpu-workers"), false},
		"single char":   {types.StringValue("a"), false},
		"max length":    {types.StringValue(strings.Repeat("a", 30)), false},
		"too long":      {types.StringValue(strings.Repeat("a", 31)), true},
		"uppercase":     {types.StringValue("Gpu"), true},
		"leading dash":  {types.StringValue("-gpu"), true},
		"trailing dash": {types.StringValue("gpu-"), true},
		"underscore":    {types.StringValue("gpu_workers"), true},
		"empty":         {types.StringValue(""), true},
		"unknown":       {types.StringUnknown(), false},
		"null":          {types.StringNull(), false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := validateManagedKubernetesString(v, tc.value); got != tc.wantErr {
				t.Fatalf("got error=%v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestManagedKubernetesCIDRValidator(t *testing.T) {
	v := managedKubernetesCIDRValidator{}
	tests := map[string]struct {
		value   string
		wantErr bool
	}{
		"pod cidr":     {"10.244.0.0/16", false},
		"/32":          {"192.168.1.1/32", false},
		"missing bits": {"10.0.0.0", true},
		"garbage":      {"not-a-cidr", true},
		"ipv6":         {"fd00::/8", true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := validateManagedKubernetesString(v, types.StringValue(tc.value)); got != tc.wantErr {
				t.Fatalf("got error=%v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestManagedKubernetesPodCIDRBitsValidator(t *testing.T) {
	v := managedKubernetesPodCIDRBitsValidator{}
	tests := map[string]struct {
		value   string
		wantErr bool
	}{
		"/16":        {"10.244.0.0/16", false},
		"/21":        {"10.244.0.0/21", false},
		"/22":        {"10.244.0.0/22", true},
		"/24":        {"10.244.0.0/24", true},
		"unparsable": {"nope", false}, // syntax errors belong to the CIDR validator
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := validateManagedKubernetesString(v, types.StringValue(tc.value)); got != tc.wantErr {
				t.Fatalf("got error=%v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestManagedKubernetesKubeconfigDurationValidator(t *testing.T) {
	v := managedKubernetesKubeconfigDurationValidator{}
	tests := map[string]struct {
		value   types.String
		wantErr bool
	}{
		"90 days":        {types.StringValue("2160h"), false},
		"1h":             {types.StringValue("1h"), false},
		"30m":            {types.StringValue("30m"), true},
		"not a duration": {types.StringValue("720 hours"), true},
		"null":           {types.StringNull(), false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := validateManagedKubernetesString(v, tc.value); got != tc.wantErr {
				t.Fatalf("got error=%v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestManagedKubernetesCompositeIDValidator(t *testing.T) {
	v := managedKubernetesCompositeIDValidator{segments: 2}
	tests := map[string]struct {
		value   string
		wantErr bool
	}{
		"two segments":   {"FIN-03/my-cluster", false},
		"one segment":    {"my-cluster", true},
		"three segments": {"a/b/c", true},
		"empty segment":  {"FIN-03/", true},
		"leading empty":  {"/my-cluster", true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := validateManagedKubernetesString(v, types.StringValue(tc.value)); got != tc.wantErr {
				t.Fatalf("got error=%v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestManagedKubernetesCIDRsOverlap(t *testing.T) {
	tests := map[string]struct {
		a, b string
		want bool
	}{
		"disjoint":        {"10.244.0.0/16", "10.96.0.0/16", false},
		"contained":       {"10.244.0.0/16", "10.244.1.0/24", true},
		"identical":       {"10.96.0.0/16", "10.96.0.0/16", true},
		"unparsable a":    {"nope", "10.96.0.0/16", false},
		"unparsable both": {"nope", "nah", false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := managedKubernetesCIDRsOverlap(tc.a, tc.b); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestManagedKubernetesNodePoolDetail(t *testing.T) {
	if got := managedKubernetesNodePoolDetail(nil); got != "" {
		t.Fatalf("nil pool: got %q", got)
	}
	if got := managedKubernetesNodePoolDetail(&verda.ManagedKubernetesNodePool{State: "creating"}); got != ` (last state "creating")` {
		t.Fatalf("no errors: got %q", got)
	}
	got := managedKubernetesNodePoolDetail(&verda.ManagedKubernetesNodePool{
		State: "degraded",
		Errors: []verda.ManagedKubernetesNodePoolError{
			{Reason: "insufficient_capacity", Message: "no capacity in FIN-03"},
		},
	})
	if !strings.Contains(got, "insufficient_capacity: no capacity in FIN-03") {
		t.Fatalf("with errors: got %q", got)
	}
}
