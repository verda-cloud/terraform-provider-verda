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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &SubnetResource{}
var _ resource.ResourceWithImportState = &SubnetResource{}

func NewSubnetResource() resource.Resource {
	return &SubnetResource{}
}

type SubnetResource struct {
	client *verda.Client
}

type SubnetResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	PrivateNetworkID types.String   `tfsdk:"private_network_id"`
	Location         types.String   `tfsdk:"location"`
	Name             types.String   `tfsdk:"name"`
	CIDR             types.String   `tfsdk:"cidr"`
	IsDefault        types.Bool     `tfsdk:"is_default"`
	CreatedAt        types.String   `tfsdk:"created_at"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

const subnetImportIDFormat = "<private_network_id>/<subnet_id>"

func (r *SubnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (r *SubnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an IPv4 subnet of a Verda private network in one location. " +
			"Subnets of the same private network must not overlap, in any location.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Subnet identifier (`sn_…`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the private network the subnet belongs to. Changing it replaces the subnet.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Location code, e.g. `FIN-03`. Must be a location where private networks are available. Changing it replaces the subnet.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the subnet, 1–64 characters, unique within the private network. Can be changed in place.",
				Validators:          privateNetworkNameValidators,
			},
			"cidr": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "IPv4 range of the subnet, e.g. `10.10.0.0/24`. Must be a network address, " +
					"`/29` or larger, and inside `10.0.0.0/8`, `172.16.0.0/12` or `192.168.0.0/16`. " +
					"Changing it replaces the subnet.",
				Validators: []validator.String{subnetCIDRValidator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_default": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether this is the private network's default subnet in its location. The first subnet " +
					"created in a location becomes its default automatically. Read-only here; " +
					"manage it with `verda_subnet_default`.",
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
			// Delete retries while instances on the subnet are still being
			// deleted (default 10m).
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Delete: true,
			}),
		},
	}
}

func (r *SubnetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SubnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubnetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := r.client.PrivateNetworks.CreateSubnet(ctx, data.PrivateNetworkID.ValueString(), verda.CreateSubnetRequest{
		LocationCode: data.Location.ValueString(),
		Name:         data.Name.ValueString(),
		CIDR:         data.CIDR.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create private network subnet, got error: %s", err))
		return
	}

	setSubnetModel(&data, subnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubnetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := r.client.PrivateNetworks.GetSubnetByID(ctx, data.PrivateNetworkID.ValueString(), data.ID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network subnet, got error: %s", err))
		return
	}

	setSubnetModel(&data, subnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SubnetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	subnet, err := r.client.PrivateNetworks.UpdateSubnet(ctx, data.PrivateNetworkID.ValueString(), data.ID.ValueString(), verda.UpdateSubnetRequest{
		Name: &name,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update private network subnet, got error: %s", err))
		return
	}

	setSubnetModel(&data, subnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubnetResourceModel
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

	privateNetworkID, id := data.PrivateNetworkID.ValueString(), data.ID.ValueString()
	err := deleteRetryingConflicts(deleteCtx, func(ctx context.Context) error {
		return r.client.PrivateNetworks.DeleteSubnet(ctx, privateNetworkID, id)
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete private network subnet %s, got error: %s", id, err))
	}
}

func (r *SubnetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseImportID(req.ID, subnetImportIDFormat)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_network_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func setSubnetModel(data *SubnetResourceModel, subnet *verda.Subnet) {
	data.ID = types.StringValue(subnet.ID)
	data.Location = types.StringValue(subnet.Location.Code)
	data.Name = types.StringValue(subnet.Name)
	data.CIDR = types.StringValue(subnet.CIDR)
	data.IsDefault = types.BoolValue(subnet.IsDefault)
	data.CreatedAt = types.StringValue(formatTimestamp(subnet.CreatedAt))
}
