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
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func TestFlattenManagedKubernetesNodePoolToModel(t *testing.T) {
	ctx := context.Background()
	r := &ManagedKubernetesNodePoolResource{}

	pool := &verda.ManagedKubernetesNodePool{
		Name:          "gpu-workers",
		Replicas:      2,
		ReadyReplicas: 1,
		InstanceType:  "1B200.30V",
		SSHKeyIDs:     []string{"key-1", "key-2"},
		State:         "creating",
		Errors: []verda.ManagedKubernetesNodePoolError{
			{
				Reason:  "insufficient_capacity",
				Message: "no 1B200.30V capacity in FIN-03",
				Since:   time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC),
			},
		},
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	data := ManagedKubernetesNodePoolResourceModel{
		ClusterID: types.StringValue("FIN-03/my-cluster"),
	}
	var diagnostics diag.Diagnostics
	r.flattenManagedKubernetesNodePoolToModel(ctx, pool, &data, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", diagnostics)
	}

	wantValues := map[string]string{
		"id":            data.ID.ValueString(),
		"name":          data.Name.ValueString(),
		"instance_type": data.InstanceType.ValueString(),
		"state":         data.State.ValueString(),
		"created_at":    data.CreatedAt.ValueString(),
	}
	want := map[string]string{
		"id":            "FIN-03/my-cluster/gpu-workers",
		"name":          "gpu-workers",
		"instance_type": "1B200.30V",
		"state":         "creating",
		"created_at":    "2026-01-02T03:04:05Z",
	}
	for field, got := range wantValues {
		if got != want[field] {
			t.Errorf("%s = %q, want %q", field, got, want[field])
		}
	}

	if data.Replicas.ValueInt64() != 2 {
		t.Errorf("replicas = %d, want 2", data.Replicas.ValueInt64())
	}
	if data.ReadyReplicas.ValueInt64() != 1 {
		t.Errorf("ready_replicas = %d, want 1", data.ReadyReplicas.ValueInt64())
	}

	var sshKeyIDs []string
	data.SSHKeyIDs.ElementsAs(ctx, &sshKeyIDs, false)
	if len(sshKeyIDs) != 2 || sshKeyIDs[0] != "key-1" || sshKeyIDs[1] != "key-2" {
		t.Errorf("ssh_key_ids = %v, want [key-1 key-2]", sshKeyIDs)
	}

	if data.Errors.IsNull() || len(data.Errors.Elements()) != 1 {
		t.Fatalf("errors = %v, want one element", data.Errors)
	}
	errorAttrs := data.Errors.Elements()[0].(types.Object).Attributes()
	if got := errorAttrs["reason"].(types.String).ValueString(); got != "insufficient_capacity" {
		t.Errorf("errors[0].reason = %q", got)
	}
	if got := errorAttrs["since"].(types.String).ValueString(); got != "2026-02-03T04:05:06Z" {
		t.Errorf("errors[0].since = %q", got)
	}
}

func TestFlattenManagedKubernetesNodePoolPreservesConfiguredSSHKeys(t *testing.T) {
	ctx := context.Background()
	r := &ManagedKubernetesNodePoolResource{}

	configuredKeys := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("key-1")})
	data := ManagedKubernetesNodePoolResourceModel{
		ClusterID: types.StringValue("FIN-03/my-cluster"),
		SSHKeyIDs: configuredKeys,
	}

	// The API omitted ssh_key_ids from the response.
	pool := &verda.ManagedKubernetesNodePool{Name: "cpu-workers", State: "active"}

	var diagnostics diag.Diagnostics
	r.flattenManagedKubernetesNodePoolToModel(ctx, pool, &data, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", diagnostics)
	}
	if !data.SSHKeyIDs.Equal(configuredKeys) {
		t.Fatalf("ssh_key_ids = %v, want preserved %v", data.SSHKeyIDs, configuredKeys)
	}
	if !data.Errors.IsNull() {
		t.Fatalf("errors = %v, want null", data.Errors)
	}
}
