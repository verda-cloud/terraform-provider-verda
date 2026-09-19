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
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestOnlyOnDestroyChanged(t *testing.T) {
	base := func() InstanceResourceModel {
		return InstanceResourceModel{
			InstanceType:    types.StringValue("CPU.8V.32G"),
			Hostname:        types.StringValue("example"),
			IsSpot:          types.BoolValue(false),
			SSHKeyIDs:       types.SetValueMust(types.StringType, []attr.Value{types.StringValue("key-1")}),
			Volumes:         types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}),
			ExistingVolumes: types.ListNull(types.StringType),
			OSVolume:        osVolumeObject(t, "example-os", 100, osVolumeDeletePermanently),
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

		got, diags := onlyOnDestroyChanged(context.Background(), plan, state)
		if diags.HasError() {
			t.Fatalf("%s: unexpected diagnostics: %v", c.description, diags)
		}
		if got != c.want {
			t.Errorf("%s: onlyOnDestroyChanged = %v, want %v", c.description, got, c.want)
		}
	}
}
