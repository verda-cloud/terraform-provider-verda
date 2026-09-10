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
