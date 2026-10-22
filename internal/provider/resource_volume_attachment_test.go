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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func TestVolumeAttachmentRead(t *testing.T) {
	target, other := "instance-target", "instance-other"
	tests := map[string]struct {
		instanceID  *string
		instances   []verda.VolumeAttachedInstance
		wantRemoved bool
	}{
		"legacy attachment": {
			instanceID: &target,
		},
		"shared primary attachment": {
			instanceID: &target,
			instances:  []verda.VolumeAttachedInstance{{ID: target}, {ID: other}},
		},
		"shared secondary attachment": {
			instanceID: &other,
			instances:  []verda.VolumeAttachedInstance{{ID: other}, {ID: target}},
		},
		"shared attachment without legacy instance ID": {
			instances: []verda.VolumeAttachedInstance{{ID: other}, {ID: target}},
		},
		"attached only to another instance": {
			instanceID:  &other,
			instances:   []verda.VolumeAttachedInstance{{ID: other}},
			wantRemoved: true,
		},
		"legacy attachment to another instance": {
			instanceID:  &other,
			wantRemoved: true,
		},
		"detached": {
			wantRemoved: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mountCommand := "mount shared-volume"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if req.Method == http.MethodPost && req.URL.Path == "/oauth2/token" {
					if err := json.NewEncoder(w).Encode(map[string]any{
						"access_token": "test-token",
						"expires_in":   3600,
						"token_type":   "Bearer",
					}); err != nil {
						t.Errorf("encoding token: %v", err)
					}
					return
				}
				if req.Method != http.MethodGet || req.URL.Path != "/volumes/volume-1" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					http.NotFound(w, req)
					return
				}
				if err := json.NewEncoder(w).Encode(verda.Volume{
					ID:           "volume-1",
					InstanceID:   tc.instanceID,
					Instances:    tc.instances,
					MountCommand: &mountCommand,
				}); err != nil {
					t.Errorf("encoding volume: %v", err)
				}
			}))
			t.Cleanup(server.Close)

			client, err := verda.NewClient(
				verda.WithBaseURL(server.URL),
				verda.WithClientID("test-client"),
				verda.WithClientSecret("test-secret"),
			)
			if err != nil {
				t.Fatalf("creating client: %v", err)
			}
			r := &VolumeAttachmentResource{client: client}
			ctx := context.Background()
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{
				Schema: schemaResp.Schema,
				Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
			}
			importResp := resource.ImportStateResponse{State: state}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "volume-1/" + target}, &importResp)
			if importResp.Diagnostics.HasError() {
				t.Fatalf("importing attachment: %v", importResp.Diagnostics)
			}

			readResp := resource.ReadResponse{State: importResp.State}
			r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("reading attachment: %v", readResp.Diagnostics)
			}
			if readResp.State.Raw.IsNull() != tc.wantRemoved {
				t.Fatalf("attachment removed = %v, want %v", readResp.State.Raw.IsNull(), tc.wantRemoved)
			}
			if tc.wantRemoved {
				return
			}
			var got VolumeAttachmentResourceModel
			if diags := readResp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("decoding attachment state: %v", diags)
			}
			if got.VolumeID.ValueString() != "volume-1" || got.InstanceID.ValueString() != target {
				t.Errorf("attachment identity changed: %s/%s", got.VolumeID, got.InstanceID)
			}
			if got.MountCommand.ValueString() != mountCommand {
				t.Errorf("mount command = %q, want %q", got.MountCommand.ValueString(), mountCommand)
			}
		})
	}
}
