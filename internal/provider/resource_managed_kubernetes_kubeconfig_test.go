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

func TestFlattenManagedKubernetesKubeconfigToModel(t *testing.T) {
	ctx := context.Background()
	r := &ManagedKubernetesKubeconfigResource{}

	expiry := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)
	record := &verda.ManagedKubernetesKubeconfig{
		ID:          "kc-123",
		DisplayName: "Terraform admin",
		CommonName:  "terraform-admin",
		Groups:      []string{"system:masters"},
		Status:      "ready",
		NotAfter:    &expiry,
		Kubeconfig:  "apiVersion: v1\nkind: Config\n",
	}

	var data ManagedKubernetesKubeconfigResourceModel
	var diagnostics diag.Diagnostics
	r.flattenManagedKubernetesKubeconfigToModel(ctx, record, &data, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", diagnostics)
	}

	wantValues := map[string]string{
		"id":           data.ID.ValueString(),
		"common_name":  data.CommonName.ValueString(),
		"display_name": data.DisplayName.ValueString(),
		"status":       data.Status.ValueString(),
		"not_after":    data.NotAfter.ValueString(),
		"content":      data.Content.ValueString(),
	}
	want := map[string]string{
		"id":           "kc-123",
		"common_name":  "terraform-admin",
		"display_name": "Terraform admin",
		"status":       "ready",
		"not_after":    "2026-05-06T07:08:09Z",
		"content":      "apiVersion: v1\nkind: Config\n",
	}
	for field, got := range wantValues {
		if got != want[field] {
			t.Errorf("%s = %q, want %q", field, got, want[field])
		}
	}

	var groups []string
	data.Groups.ElementsAs(ctx, &groups, false)
	if len(groups) != 1 || groups[0] != "system:masters" {
		t.Errorf("groups = %v, want [system:masters]", groups)
	}
}

func TestFlattenManagedKubernetesKubeconfigPendingPreservesStateFields(t *testing.T) {
	ctx := context.Background()
	r := &ManagedKubernetesKubeconfigResource{}

	// Simulate a Read while the record is still pending: the API omits
	// content, not_after and groups.
	data := ManagedKubernetesKubeconfigResourceModel{
		Content:     types.StringValue("existing-content"),
		NotAfter:    types.StringValue("2026-05-06T07:08:09Z"),
		Groups:      types.ListValueMust(types.StringType, []attr.Value{types.StringValue("system:masters")}),
		DisplayName: types.StringValue("Terraform admin"),
	}

	record := &verda.ManagedKubernetesKubeconfig{
		ID:         "kc-123",
		CommonName: "terraform-admin",
		Status:     "pending",
	}

	var diagnostics diag.Diagnostics
	r.flattenManagedKubernetesKubeconfigToModel(ctx, record, &data, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", diagnostics)
	}

	if got := data.Content.ValueString(); got != "existing-content" {
		t.Errorf("content = %q, want preserved", got)
	}
	if got := data.NotAfter.ValueString(); got != "2026-05-06T07:08:09Z" {
		t.Errorf("not_after = %q, want preserved", got)
	}
	var groups []string
	data.Groups.ElementsAs(ctx, &groups, false)
	if len(groups) != 1 || groups[0] != "system:masters" {
		t.Errorf("groups = %v, want preserved", groups)
	}
	if data.DisplayName.ValueString() != "Terraform admin" {
		t.Errorf("display_name = %q, want preserved", data.DisplayName.ValueString())
	}
	if data.Status.ValueString() != "pending" {
		t.Errorf("status = %q, want pending", data.Status.ValueString())
	}
}
