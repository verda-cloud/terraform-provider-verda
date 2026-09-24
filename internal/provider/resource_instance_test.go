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
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func TestSetCreateRequestSSHKeyIDs(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		value   types.Set
		want    []string
		wantNil bool
	}{
		"unknown": {
			value:   types.SetUnknown(types.StringType),
			wantNil: true,
		},
		"empty": {
			value: types.SetValueMust(types.StringType, []attr.Value{}),
			want:  []string{},
		},
		"configured": {
			value: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("key-1")}),
			want:  []string{"key-1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var diagnostics diag.Diagnostics
			createReq := verda.CreateInstanceRequest{}

			setCreateRequestSSHKeyIDs(ctx, tc.value, &createReq, &diagnostics)

			if diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diagnostics)
			}
			if tc.wantNil {
				if createReq.SSHKeyIDs != nil {
					t.Fatalf("expected nil SSHKeyIDs, got %v", createReq.SSHKeyIDs)
				}
				return
			}
			if createReq.SSHKeyIDs == nil {
				t.Fatal("expected non-nil SSHKeyIDs")
			}
			if !sameStringElements(createReq.SSHKeyIDs, tc.want) {
				t.Fatalf("expected SSHKeyIDs %v, got %v", tc.want, createReq.SSHKeyIDs)
			}
		})
	}
}

func TestPreserveKnownSSHKeyIDsAfterCreate(t *testing.T) {
	plannedSSHKeyIDs := types.SetValueMust(types.StringType, []attr.Value{})
	data := InstanceResourceModel{
		SSHKeyIDs: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("api-key")}),
	}

	preserveKnownSSHKeyIDs(plannedSSHKeyIDs, &data)

	assertSetStrings(t, data.SSHKeyIDs, []string{})
}

func TestPreserveKnownSSHKeyIDsAfterRead(t *testing.T) {
	priorSSHKeyIDs := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("configured-key")})
	data := InstanceResourceModel{
		SSHKeyIDs: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("api-key")}),
	}

	preserveKnownSSHKeyIDs(priorSSHKeyIDs, &data)

	assertSetStrings(t, data.SSHKeyIDs, []string{"configured-key"})
}

func TestPreserveKnownImage(t *testing.T) {
	image := types.StringValue("79091d37-cf39-43cd-b193-ba9c2611c988")
	data := InstanceResourceModel{
		Image: types.StringValue("ubuntu-24.04"),
	}

	preserveKnownImage(image, &data)

	if data.Image.ValueString() != image.ValueString() {
		t.Fatalf("expected image %q, got %q", image.ValueString(), data.Image.ValueString())
	}
}

func assertSetStrings(t *testing.T, set types.Set, want []string) {
	t.Helper()

	var got []string
	diags := set.ElementsAs(context.Background(), &got, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !sameStringElements(got, want) {
		t.Fatalf("expected set %v, got %v", want, got)
	}
}

func sameStringElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := append([]string(nil), a...)
	bCopy := append([]string(nil), b...)
	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for index := range aCopy {
		if aCopy[index] != bCopy[index] {
			return false
		}
	}

	return true
}

func TestDeleteVolumePolicy(t *testing.T) {
	cases := []struct {
		onDestroy   string
		wantIDs     []string
		wantPerm    bool
		description string
	}{
		{"", nil, true, "unset: the API default deletes the OS volume, permanently"},
		{osVolumeDeletePermanently, nil, true, "delete_permanently: API default, no trash"},
		{osVolumeMoveToTrash, nil, false, "move_to_trash: API default, trash first"},
		{osVolumeKeepDetached, []string{}, false, "keep_detached: an empty list deletes no volume"},
	}
	for _, c := range cases {
		ids, perm := deleteVolumePolicy(c.onDestroy)
		if perm != c.wantPerm {
			t.Errorf("%s: delete_permanently = %v, want %v", c.description, perm, c.wantPerm)
		}
		if (ids == nil) != (c.wantIDs == nil) || len(ids) != len(c.wantIDs) {
			t.Errorf("%s: volume_ids = %#v, want %#v", c.description, ids, c.wantIDs)
		}
	}
}

var osVolumeAttrTypes = map[string]attr.Type{
	"name":                types.StringType,
	"size":                types.Int64Type,
	"type":                types.StringType,
	"on_spot_discontinue": types.StringType,
	"on_destroy":          types.StringType,
}

func osVolumeObject(t *testing.T, name string, size int64, onDestroy string) types.Object {
	t.Helper()

	object, diags := types.ObjectValueFrom(context.Background(), osVolumeAttrTypes, OSVolumeCreateModel{
		Name:              types.StringValue(name),
		Size:              types.Int64Value(size),
		Type:              types.StringValue("NVMe"),
		OnSpotDiscontinue: types.StringNull(),
		OnDestroy:         types.StringValue(onDestroy),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	return object
}

func TestOnlyProviderSettingsChanged(t *testing.T) {
	base := func() InstanceResourceModel {
		return InstanceResourceModel{
			InstanceType:    types.StringValue("CPU.8V.32G"),
			Hostname:        types.StringValue("example"),
			IsSpot:          types.BoolValue(false),
			SSHKeyIDs:       types.SetValueMust(types.StringType, []attr.Value{types.StringValue("key-1")}),
			Volumes:         types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}),
			ExistingVolumes: types.ListNull(types.StringType),
			OSVolume:        osVolumeObject(t, "example-os", 100, osVolumeDeletePermanently),
			Timeouts:        createTimeoutValue("20m"),
			// Computed attributes are unknown in an update plan and must not
			// count as a change.
			Status: types.StringUnknown(),
		}
	}

	cases := []struct {
		description string
		mutate      func(plan *InstanceResourceModel)
		want        bool
	}{
		{
			"on_destroy change is the only difference",
			func(plan *InstanceResourceModel) {
				plan.OSVolume = osVolumeObject(t, "example-os", 100, osVolumeKeepDetached)
			},
			true,
		},
		{
			"timeouts change is the only difference",
			func(plan *InstanceResourceModel) {
				plan.Timeouts = createTimeoutValue("45m")
			},
			true,
		},
		{
			"on_destroy and timeouts changed together",
			func(plan *InstanceResourceModel) {
				plan.OSVolume = osVolumeObject(t, "example-os", 100, osVolumeKeepDetached)
				plan.Timeouts = createTimeoutValue("45m")
			},
			true,
		},
		{
			"hostname changed alongside timeouts",
			func(plan *InstanceResourceModel) {
				plan.Hostname = types.StringValue("renamed")
				plan.Timeouts = createTimeoutValue("45m")
			},
			false,
		},
		{
			"no change at all",
			func(plan *InstanceResourceModel) {},
			true,
		},
		{
			"os_volume size changed alongside on_destroy",
			func(plan *InstanceResourceModel) {
				plan.OSVolume = osVolumeObject(t, "example-os", 200, osVolumeKeepDetached)
			},
			false,
		},
		{
			"os_volume removed",
			func(plan *InstanceResourceModel) {
				plan.OSVolume = types.ObjectNull(osVolumeAttrTypes)
			},
			false,
		},
		{
			"is_spot changed",
			func(plan *InstanceResourceModel) {
				plan.IsSpot = types.BoolValue(true)
			},
			false,
		},
	}
	for _, c := range cases {
		state := base()
		plan := base()
		plan.Status = types.StringValue("running")
		c.mutate(&plan)

		got, diags := onlyProviderSettingsChanged(context.Background(), plan, state)
		if diags.HasError() {
			t.Fatalf("%s: unexpected diagnostics: %v", c.description, diags)
		}
		if got != c.want {
			t.Errorf("%s: onlyProviderSettingsChanged = %v, want %v", c.description, got, c.want)
		}
	}
}

func createTimeoutValue(create string) timeouts.Value {
	return timeouts.Value{Object: types.ObjectValueMust(
		map[string]attr.Type{"create": types.StringType},
		map[string]attr.Value{"create": types.StringValue(create)},
	)}
}

// A timeouts-only change reaches Update as an in-place update; Update must
// store the new value and keep the instance as it is.
func TestInstanceUpdateStoresTimeouts(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&InstanceResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	build := func(create string, status attr.Value) tftypes.Value {
		st := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		var diags diag.Diagnostics
		diags.Append(st.SetAttribute(ctx, path.Root("id"), "i-1")...)
		diags.Append(st.SetAttribute(ctx, path.Root("hostname"), "example")...)
		diags.Append(st.SetAttribute(ctx, path.Root("ip"), "203.0.113.7")...)
		diags.Append(st.SetAttribute(ctx, path.Root("is_spot"), false)...)
		diags.Append(st.SetAttribute(ctx, path.Root("status"), status)...)
		diags.Append(st.SetAttribute(ctx, path.Root("timeouts").AtName("create"), create)...)
		if diags.HasError() {
			t.Fatalf("building state: %v", diags)
		}
		return st.Raw
	}
	// The plan for an in-place update leaves computed attributes unknown.
	planned := build("45m", types.StringUnknown())
	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s, Raw: planned},
		State: tfsdk.State{Schema: s, Raw: build("20m", types.StringValue(verda.StatusRunning))},
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: planned}}
	(&InstanceResource{}).Update(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("timeouts-only update failed: %v", resp.Diagnostics)
	}

	var create, status, ip types.String
	resp.State.GetAttribute(ctx, path.Root("timeouts").AtName("create"), &create)
	resp.State.GetAttribute(ctx, path.Root("status"), &status)
	resp.State.GetAttribute(ctx, path.Root("ip"), &ip)
	if create.ValueString() != "45m" {
		t.Errorf("timeouts.create = %q, want the planned 45m", create.ValueString())
	}
	if status.ValueString() != verda.StatusRunning || ip.ValueString() != "203.0.113.7" {
		t.Errorf("status = %s, ip = %s; want the prior state kept", status, ip)
	}
}

// fakeInstances answers GetByID from a scripted sequence of states.
type fakeInstances struct {
	steps []fakeStep
	calls int
}

type fakeStep struct {
	status string
	ip     string
	err    error
	hang   bool // block until ctx is done, like a request the server never answers
}

func (f *fakeInstances) GetByID(ctx context.Context, id string) (*verda.Instance, error) {
	step := f.steps[len(f.steps)-1]
	if f.calls < len(f.steps) {
		step = f.steps[f.calls]
	}
	f.calls++
	if step.hang {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if step.err != nil {
		return nil, step.err
	}
	inst := &verda.Instance{ID: id, Status: step.status}
	if step.ip != "" {
		ip := step.ip
		inst.IP = &ip
	}
	return inst, nil
}

func TestWaitForRunning(t *testing.T) {
	instancePollInterval = time.Millisecond
	t.Cleanup(func() { instancePollInterval = 10 * time.Second })
	withTimeout := func(t *testing.T, d time.Duration) context.Context {
		ctx, cancel := context.WithTimeout(context.Background(), d)
		t.Cleanup(cancel)
		return ctx
	}

	t.Run("provisioning then running with an address", func(t *testing.T) {
		f := &fakeInstances{steps: []fakeStep{
			{status: verda.StatusProvisioning},
			{status: verda.StatusRunning}, // running, no address yet
			{status: verda.StatusRunning, ip: "203.0.113.7"},
		}}
		inst, err := waitForRunning(withTimeout(t, time.Second), f, "i-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inst.IP == nil || *inst.IP != "203.0.113.7" {
			t.Fatalf("ip = %v, want 203.0.113.7", inst.IP)
		}
		if f.calls != 3 {
			t.Fatalf("calls = %d, want 3", f.calls)
		}
	})

	t.Run("a read error is retried", func(t *testing.T) {
		f := &fakeInstances{steps: []fakeStep{
			{err: errors.New("502")},
			{status: verda.StatusRunning, ip: "203.0.113.8"},
		}}
		if _, err := waitForRunning(withTimeout(t, time.Second), f, "i-2"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a terminal status fails at once", func(t *testing.T) {
		f := &fakeInstances{steps: []fakeStep{{status: verda.StatusError}}}
		_, err := waitForRunning(withTimeout(t, time.Second), f, "i-3")
		if err == nil || f.calls != 1 {
			t.Fatalf("err = %v, calls = %d; want an error after one call", err, f.calls)
		}
	})

	t.Run("the timeout names the last status", func(t *testing.T) {
		f := &fakeInstances{steps: []fakeStep{{status: verda.StatusProvisioning}}}
		_, err := waitForRunning(withTimeout(t, 5*time.Millisecond), f, "i-4")
		if err == nil || !strings.Contains(err.Error(), verda.StatusProvisioning) {
			t.Fatalf("err = %v; want a timeout naming %q", err, verda.StatusProvisioning)
		}
	})

	t.Run("the deadline interrupts an in-flight read", func(t *testing.T) {
		f := &fakeInstances{steps: []fakeStep{
			{status: verda.StatusProvisioning},
			{hang: true},
		}}
		start := time.Now()
		_, err := waitForRunning(withTimeout(t, 20*time.Millisecond), f, "i-5")
		if err == nil || !strings.Contains(err.Error(), verda.StatusProvisioning) {
			t.Fatalf("err = %v; want a timeout naming %q", err, verda.StatusProvisioning)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("returned after %s; the hung read was not cancelled", elapsed)
		}
	})

	t.Run("a cancelled apply is not reported as a timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f := &fakeInstances{steps: []fakeStep{{status: verda.StatusProvisioning}}}
		_, err := waitForRunning(ctx, f, "i-6")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v; want context.Canceled", err)
		}
	})
}

func TestPreserveKnownIP(t *testing.T) {
	prior := types.StringValue("203.0.113.9")
	running := &verda.Instance{Status: verda.StatusRunning}
	data := InstanceResourceModel{IP: types.StringNull()}
	preserveKnownIP(prior, running, &data)
	if data.IP.ValueString() != "203.0.113.9" {
		t.Fatalf("running instance with no ip from the API: ip = %q, want the prior one", data.IP.ValueString())
	}

	stopped := &verda.Instance{Status: verda.StatusOffline}
	data = InstanceResourceModel{IP: types.StringNull()}
	preserveKnownIP(prior, stopped, &data)
	if !data.IP.IsNull() {
		t.Fatalf("offline instance: ip = %q, want null", data.IP.ValueString())
	}
}
