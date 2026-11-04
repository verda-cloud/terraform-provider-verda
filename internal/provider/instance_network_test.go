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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func networkObject(t *testing.T, subnetID, privateIP, publicIP string) types.Object {
	t.Helper()
	v, diags := types.ObjectValue(instanceNetworkAttrTypes, map[string]attr.Value{
		"subnet_id":  types.StringValue(subnetID),
		"private_ip": types.StringValue(privateIP),
		"public_ip":  types.StringValue(publicIP),
	})
	if diags.HasError() {
		t.Fatalf("network object: %v", diags)
	}
	return v
}

func strPtr(s string) *string { return &s }

func TestSetCreateRequestNetwork(t *testing.T) {
	ctx := context.Background()

	t.Run("no block leaves the request alone", func(t *testing.T) {
		var req verda.CreateInstanceRequest
		var diags diag.Diagnostics
		got := setCreateRequestNetwork(ctx, types.ObjectNull(instanceNetworkAttrTypes), &req, &diags)
		if got != nil || diags.HasError() || req.Subnet != "" || req.PrivateIP != nil || req.PublicIP != nil {
			t.Fatalf("got %+v, req %+v, diags %v", got, req, diags)
		}
	})

	t.Run("private-only with a fixed address", func(t *testing.T) {
		var req verda.CreateInstanceRequest
		var diags diag.Diagnostics
		got := setCreateRequestNetwork(ctx, networkObject(t, "sn_1", "10.0.0.5", "none"), &req, &diags)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if req.Subnet != "sn_1" || req.Network != "" {
			t.Errorf("subnet = %q, network = %q", req.Subnet, req.Network)
		}
		if req.PrivateIP == nil || *req.PrivateIP != "10.0.0.5" || req.PublicIP == nil || *req.PublicIP != verda.IPAllocNone {
			t.Errorf("private = %v, public = %v", req.PrivateIP, req.PublicIP)
		}
		if addressesFor(got) != (instanceAddresses{private: true}) {
			t.Errorf("addresses = %+v", addressesFor(got))
		}
	})

	t.Run("null allocations default to auto and are sent together", func(t *testing.T) {
		obj, _ := types.ObjectValue(instanceNetworkAttrTypes, map[string]attr.Value{
			"subnet_id":  types.StringValue("sn_1"),
			"private_ip": types.StringNull(),
			"public_ip":  types.StringNull(),
		})
		var req verda.CreateInstanceRequest
		var diags diag.Diagnostics
		got := setCreateRequestNetwork(ctx, obj, &req, &diags)
		if req.PrivateIP == nil || *req.PrivateIP != verda.IPAllocAuto || req.PublicIP == nil || *req.PublicIP != verda.IPAllocAuto {
			t.Errorf("private = %v, public = %v", req.PrivateIP, req.PublicIP)
		}
		if addressesFor(got) != (instanceAddresses{public: true, private: true}) {
			t.Errorf("addresses = %+v", addressesFor(got))
		}
	})
}

func TestAddressesForLegacy(t *testing.T) {
	if addressesFor(nil) != legacyAddresses {
		t.Fatalf("addressesFor(nil) = %+v", addressesFor(nil))
	}
}

func TestInstanceAddressesAssigned(t *testing.T) {
	both := &verda.Instance{IP: strPtr("203.0.113.1"), PrivateIP: strPtr("10.0.0.2")}
	publicOnly := &verda.Instance{IP: strPtr("203.0.113.1")}
	privateOnly := &verda.Instance{PrivateIP: strPtr("10.0.0.2")}
	emptyStrings := &verda.Instance{IP: strPtr(""), PrivateIP: strPtr("")}

	tests := []struct {
		name string
		want instanceAddresses
		inst *verda.Instance
		ok   bool
	}{
		{"legacy has public", legacyAddresses, publicOnly, true},
		{"legacy lacks public", legacyAddresses, privateOnly, false},
		{"private network auto has both", instanceAddresses{public: true, private: true}, both, true},
		{"private network auto lacks public", instanceAddresses{public: true, private: true}, privateOnly, false},
		{"private network auto lacks private", instanceAddresses{public: true, private: true}, publicOnly, false},
		{"private network private-only has private", instanceAddresses{private: true}, privateOnly, true},
		{"empty strings do not count", instanceAddresses{public: true, private: true}, emptyStrings, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.want.assigned(tt.inst); got != tt.ok {
				t.Errorf("assigned = %v, want %v", got, tt.ok)
			}
		})
	}
}

// privateIPInstances returns running instances that only ever get a private IP.
type privateIPInstances struct{ calls int }

func (f *privateIPInstances) GetByID(ctx context.Context, id string) (*verda.Instance, error) {
	f.calls++
	inst := &verda.Instance{ID: id, Status: verda.StatusRunning}
	if f.calls > 1 {
		inst.PrivateIP = strPtr("10.0.0.9")
	}
	return inst, nil
}

func TestWaitForRunningPrivateOnly(t *testing.T) {
	instancePollInterval = time.Millisecond
	t.Cleanup(func() { instancePollInterval = 10 * time.Second })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	f := &privateIPInstances{}
	inst, err := waitForRunning(ctx, f, "i-1", instanceAddresses{private: true})
	if err != nil || inst.PrivateIP == nil || f.calls != 2 {
		t.Fatalf("inst = %+v, err = %v, calls = %d", inst, err, f.calls)
	}

	// The same instance never satisfies a wait for a public address.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	if _, err := waitForRunning(ctx2, &privateIPInstances{}, "i-2", legacyAddresses); err == nil {
		t.Fatal("expected a timeout waiting for a public address")
	}
}

func TestRefreshNetwork(t *testing.T) {
	ctx := context.Background()

	t.Run("no block in state stays null even on a private network instance", func(t *testing.T) {
		data := InstanceResourceModel{Network: types.ObjectNull(instanceNetworkAttrTypes)}
		diags := refreshNetwork(ctx, &verda.Instance{SubnetID: strPtr("sn_1")}, &data)
		if diags.HasError() || !data.Network.IsNull() {
			t.Fatalf("network = %v, diags = %v", data.Network, diags)
		}
	})

	t.Run("subnet follows the API, allocations stay as configured", func(t *testing.T) {
		data := InstanceResourceModel{Network: networkObject(t, "sn_old", "auto", "none")}
		inst := &verda.Instance{SubnetID: strPtr("sn_new"), PrivateIP: strPtr("10.0.0.7")}
		if diags := refreshNetwork(ctx, inst, &data); diags.HasError() {
			t.Fatal(diags)
		}
		var got InstanceNetworkModel
		data.Network.As(ctx, &got, basetypes.ObjectAsOptions{})
		if got.SubnetID.ValueString() != "sn_new" || got.PrivateIP.ValueString() != "auto" || got.PublicIP.ValueString() != "none" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("missing subnet in the API keeps the state value", func(t *testing.T) {
		data := InstanceResourceModel{Network: networkObject(t, "sn_1", "auto", "auto")}
		if diags := refreshNetwork(ctx, &verda.Instance{}, &data); diags.HasError() {
			t.Fatal(diags)
		}
		if !data.Network.Equal(networkObject(t, "sn_1", "auto", "auto")) {
			t.Fatalf("network = %v", data.Network)
		}
	})
}

func TestImportedNetwork(t *testing.T) {
	tests := []struct {
		name string
		inst *verda.Instance
		want types.Object
	}{
		{"legacy instance", &verda.Instance{IP: strPtr("203.0.113.1")}, types.ObjectNull(instanceNetworkAttrTypes)},
		{"empty subnet id", &verda.Instance{SubnetID: strPtr("")}, types.ObjectNull(instanceNetworkAttrTypes)},
		{"private network with public ip", &verda.Instance{SubnetID: strPtr("sn_1"), IP: strPtr("203.0.113.1")}, networkObject(t, "sn_1", "auto", "auto")},
		{"private network private-only", &verda.Instance{SubnetID: strPtr("sn_1"), PrivateIP: strPtr("10.0.0.2")}, networkObject(t, "sn_1", "auto", "none")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := importedNetwork(tt.inst)
			if diags.HasError() || !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v (diags %v)", got, tt.want, diags)
			}
		})
	}
}

func TestOnlyProviderSettingsChangedNetwork(t *testing.T) {
	ctx := context.Background()
	state := InstanceResourceModel{
		InstanceType:    types.StringValue("CPU.8V.32G"),
		SSHKeyIDs:       types.SetNull(types.StringType),
		Volumes:         types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}),
		ExistingVolumes: types.ListNull(types.StringType),
		OSVolume:        types.ObjectNull(map[string]attr.Type{"name": types.StringType}),
		Network:         networkObject(t, "sn_1", "auto", "auto"),
	}
	plan := state
	plan.Network = networkObject(t, "sn_2", "auto", "auto")
	if ok, _ := onlyProviderSettingsChanged(ctx, plan, state); ok {
		t.Fatal("a network change must not count as a provider-only change")
	}
	plan.Network = state.Network
	if ok, _ := onlyProviderSettingsChanged(ctx, plan, state); !ok {
		t.Fatal("identical network must count as unchanged")
	}

	// After an import, state has private_ip "auto" while the config pins the
	// address the instance already has: no replacement, only a state update.
	state.PrivateIPAddr = types.StringValue("10.0.0.50")
	plan.Network = networkObject(t, "sn_1", "10.0.0.50", "auto")
	if ok, _ := onlyProviderSettingsChanged(ctx, plan, state); !ok {
		t.Fatal("auto -> assigned address must count as a provider-only change")
	}
	plan.Network = networkObject(t, "sn_1", "10.0.0.51", "auto")
	if ok, _ := onlyProviderSettingsChanged(ctx, plan, state); ok {
		t.Fatal("auto -> a different address must not count as a provider-only change")
	}
}

func TestPrivateIPEquivalent(t *testing.T) {
	auto, addr, other := types.StringValue("auto"), types.StringValue("10.0.0.5"), types.StringValue("10.0.0.6")
	assigned := types.StringValue("10.0.0.5")
	tests := []struct {
		name     string
		a, b     types.String
		assigned types.String
		want     bool
	}{
		{"equal auto", auto, auto, assigned, true},
		{"equal literal", addr, addr, types.StringNull(), true},
		{"auto to assigned", auto, addr, assigned, true},
		{"assigned to auto", addr, auto, assigned, true},
		{"auto to other address", auto, other, assigned, false},
		{"literal to literal", addr, other, assigned, false},
		{"assigned unknown", auto, addr, types.StringNull(), false},
		{"plan unknown", types.StringUnknown(), auto, assigned, false},
		{"assigned address still unknown", auto, addr, types.StringUnknown(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := privateIPEquivalent(tt.a, tt.b, tt.assigned); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetworkEquivalent(t *testing.T) {
	ctx := context.Background()
	assigned := types.StringValue("10.0.0.5")
	null := types.ObjectNull(instanceNetworkAttrTypes)
	tests := []struct {
		name        string
		plan, state types.Object
		want        bool
	}{
		{"both null", null, null, true},
		{"added", networkObject(t, "sn_1", "auto", "auto"), null, false},
		{"removed", null, networkObject(t, "sn_1", "auto", "auto"), false},
		{"auto vs assigned", networkObject(t, "sn_1", "10.0.0.5", "auto"), networkObject(t, "sn_1", "auto", "auto"), true},
		{"subnet changed", networkObject(t, "sn_2", "auto", "auto"), networkObject(t, "sn_1", "auto", "auto"), false},
		{"public changed", networkObject(t, "sn_1", "auto", "none"), networkObject(t, "sn_1", "auto", "auto"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := networkEquivalent(ctx, tt.plan, tt.state, assigned)
			if diags.HasError() || got != tt.want {
				t.Errorf("got %v (diags %v), want %v", got, diags, tt.want)
			}
		})
	}
}
