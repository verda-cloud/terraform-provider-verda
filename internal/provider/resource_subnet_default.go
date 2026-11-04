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

var _ resource.Resource = &SubnetDefaultResource{}
var _ resource.ResourceWithImportState = &SubnetDefaultResource{}

func NewSubnetDefaultResource() resource.Resource {
	return &SubnetDefaultResource{}
}

type SubnetDefaultResource struct {
	client *verda.Client
}

type SubnetDefaultResourceModel struct {
	ID               types.String `tfsdk:"id"`
	PrivateNetworkID types.String `tfsdk:"private_network_id"`
	Location         types.String `tfsdk:"location"`
	SubnetID         types.String `tfsdk:"subnet_id"`
}

const subnetDefaultImportIDFormat = "<private_network_id>/<location>"

func (r *SubnetDefaultResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet_default"
}

func (r *SubnetDefaultResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sets which subnet is a private network's default in one location. Declare at most one per private network " +
			"and location. Destroying this resource only removes it from Terraform state: the private network keeps its " +
			"current default subnet.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "`<private_network_id>/<location>`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the private network. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Location code, e.g. `FIN-03`. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_id": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "ID of the subnet to make the default. It must belong to `private_network_id` and be " +
					"in `location`. Can be changed in place.",
			},
		},
	}
}

func (r *SubnetDefaultResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// set checks the subnet is in the configured location, then marks it the
// default; the API clears the previous default in that location.
func (r *SubnetDefaultResource) set(ctx context.Context, data *SubnetDefaultResourceModel) error {
	privateNetworkID, subnetID, location := data.PrivateNetworkID.ValueString(), data.SubnetID.ValueString(), data.Location.ValueString()

	subnet, err := r.client.PrivateNetworks.GetSubnetByID(ctx, privateNetworkID, subnetID)
	if err != nil {
		return err
	}
	if subnet.Location.Code != location {
		return fmt.Errorf("subnet %s is in %s, not %s", subnetID, subnet.Location.Code, location)
	}

	isDefault := true
	if _, err := r.client.PrivateNetworks.UpdateSubnet(ctx, privateNetworkID, subnetID, verda.UpdateSubnetRequest{
		IsDefault: &isDefault,
	}); err != nil {
		return err
	}
	data.ID = types.StringValue(privateNetworkID + "/" + location)
	return nil
}

func (r *SubnetDefaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubnetDefaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.set(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set the default subnet, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetDefaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubnetDefaultResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnets, err := r.client.PrivateNetworks.GetSubnets(ctx, data.PrivateNetworkID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list private network subnets, got error: %s", err))
		return
	}

	// A default changed outside Terraform shows up as drift on subnet_id.
	// With no default in the location, the next apply sets one again.
	subnet, err := findSubnet(subnets, "", data.Location.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.ID = types.StringValue(data.PrivateNetworkID.ValueString() + "/" + data.Location.ValueString())
	data.SubnetID = types.StringValue(subnet.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetDefaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SubnetDefaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.set(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set the default subnet, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetDefaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Default subnet left unchanged",
		"verda_subnet_default was removed from Terraform state only. The private network keeps its current default subnet in this location.")
}

func (r *SubnetDefaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseImportID(req.ID, subnetDefaultImportIDFormat)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_network_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("location"), parts[1])...)
}
