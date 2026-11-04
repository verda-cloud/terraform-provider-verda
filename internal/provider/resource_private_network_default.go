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
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &PrivateNetworkDefaultResource{}
var _ resource.ResourceWithImportState = &PrivateNetworkDefaultResource{}

func NewPrivateNetworkDefaultResource() resource.Resource {
	return &PrivateNetworkDefaultResource{}
}

type PrivateNetworkDefaultResource struct {
	client *verda.Client
}

type PrivateNetworkDefaultResourceModel struct {
	ID               types.String `tfsdk:"id"`
	PrivateNetworkID types.String `tfsdk:"private_network_id"`
}

// privateNetworkDefaultID is the fixed ID of the project's single default private network setting.
const privateNetworkDefaultID = "default"

func (r *PrivateNetworkDefaultResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network_default"
}

func (r *PrivateNetworkDefaultResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sets which private network is the project's default. Declare at most one per project. " +
			"Destroying this resource only removes it from Terraform state: the project keeps its current default.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Always `default`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the private network to make the project default. Can be changed in place.",
			},
		},
	}
}

func (r *PrivateNetworkDefaultResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*verda.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *verda.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// set marks the private network as the project default; the API clears the previous one.
func (r *PrivateNetworkDefaultResource) set(ctx context.Context, data *PrivateNetworkDefaultResourceModel) error {
	isDefault := true
	_, err := r.client.PrivateNetworks.Update(ctx, data.PrivateNetworkID.ValueString(), verda.UpdatePrivateNetworkRequest{
		IsDefault: &isDefault,
	})
	if err != nil {
		return err
	}
	data.ID = types.StringValue(privateNetworkDefaultID)
	return nil
}

func (r *PrivateNetworkDefaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateNetworkDefaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.set(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set the default private network, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkDefaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateNetworkDefaultResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networks, err := r.client.PrivateNetworks.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list private networks, got error: %s", err))
		return
	}

	// A default changed outside Terraform shows up as drift on private_network_id. With
	// no default at all, the next apply sets one again.
	network, err := findPrivateNetwork(networks, "")
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.ID = types.StringValue(privateNetworkDefaultID)
	data.PrivateNetworkID = types.StringValue(network.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkDefaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PrivateNetworkDefaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.set(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set the default private network, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkDefaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Default private network left unchanged",
		"verda_private_network_default was removed from Terraform state only. The project keeps its current default private network.")
}

func (r *PrivateNetworkDefaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != privateNetworkDefaultID {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("expected import ID %q, got %q", privateNetworkDefaultID, req.ID))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
