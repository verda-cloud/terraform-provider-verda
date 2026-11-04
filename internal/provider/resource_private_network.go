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

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &PrivateNetworkResource{}
var _ resource.ResourceWithImportState = &PrivateNetworkResource{}

func NewPrivateNetworkResource() resource.Resource {
	return &PrivateNetworkResource{}
}

type PrivateNetworkResource struct {
	client *verda.Client
}

type PrivateNetworkResourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	Mode      types.String   `tfsdk:"mode"`
	IsDefault types.Bool     `tfsdk:"is_default"`
	CreatedAt types.String   `tfsdk:"created_at"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

func (r *PrivateNetworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network"
}

func (r *PrivateNetworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda private network. " +
			"A private network is global; add subnets to it per location with `verda_subnet`. " +
			"Private networks created by Terraform are always custom mode (subnets are created explicitly).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Private network identifier (`pn_…`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the private network, 1–64 characters, unique within the project. Can be changed in place.",
				Validators:          privateNetworkNameValidators,
			},
			"mode": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Subnet allocation mode. Always `custom` for private networks created by Terraform; " +
					"`auto` for an imported network the platform created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_default": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether this is the project's default private network. Read-only here; " +
					"manage it with `verda_private_network_default`.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Delete retries while instances on the private network are still being
			// deleted (default 10m).
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Delete: true,
			}),
		},
	}
}

func (r *PrivateNetworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrivateNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, err := r.client.PrivateNetworks.Create(ctx, verda.CreatePrivateNetworkRequest{
		Name: data.Name.ValueString(),
		Mode: verda.NetworkModeCustom,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create private network, got error: %s", err))
		return
	}

	setPrivateNetworkModel(&data, network)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, err := r.client.PrivateNetworks.GetByID(ctx, data.ID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network, got error: %s", err))
		return
	}

	setPrivateNetworkModel(&data, network)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PrivateNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	network, err := r.client.PrivateNetworks.Update(ctx, data.ID.ValueString(), verda.UpdatePrivateNetworkRequest{
		Name: &name,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update private network, got error: %s", err))
		return
	}

	setPrivateNetworkModel(&data, network)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := data.Timeouts.Delete(ctx, defaultPrivateNetworkDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id := data.ID.ValueString()
	err := deleteRetryingConflicts(deleteCtx, func(ctx context.Context) error {
		return r.client.PrivateNetworks.Delete(ctx, id)
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete private network %s, got error: %s", id, err))
	}
}

func (r *PrivateNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setPrivateNetworkModel(data *PrivateNetworkResourceModel, network *verda.PrivateNetwork) {
	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)
	data.Mode = types.StringValue(network.Mode)
	data.IsDefault = types.BoolValue(network.IsDefault)
	data.CreatedAt = types.StringValue(formatTimestamp(network.CreatedAt))
}
