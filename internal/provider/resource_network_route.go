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

var _ resource.Resource = &NetworkRouteResource{}
var _ resource.ResourceWithImportState = &NetworkRouteResource{}

func NewNetworkRouteResource() resource.Resource {
	return &NetworkRouteResource{}
}

type NetworkRouteResource struct {
	client *verda.Client
}

type NetworkRouteResourceModel struct {
	ID               types.String `tfsdk:"id"`
	PrivateNetworkID types.String `tfsdk:"private_network_id"`
	SubnetID         types.String `tfsdk:"subnet_id"`
	Destination      types.String `tfsdk:"destination"`
	InstanceID       types.String `tfsdk:"instance_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

const routeImportIDFormat = "<private_network_id>/<subnet_id>/<route_id>"

func (r *NetworkRouteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_route"
}

func (r *NetworkRouteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Routes a private network subnet's traffic for a destination through an instance, for example " +
			"a NAT instance that gives private-only instances internet access. A subnet has at most one route " +
			"per destination: if one already exists (for example made in the console), creating this resource " +
			"takes it over and points it at `instance_id`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Route identifier (`rt_…`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the private network. Changing it replaces the route.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the subnet whose traffic is routed. Changing it replaces the route.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destination": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Destination IPv4 CIDR block. Currently the API accepts only `0.0.0.0/0`. " +
					"Changing it replaces the route.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "ID of the instance to route through. It must be on the same private network, in the " +
					"subnet's location, and have a public IP. Can be changed in place.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NetworkRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// upsert creates the route, or moves the subnet's existing route for the
// destination to the instance.
func (r *NetworkRouteResource) upsert(ctx context.Context, data *NetworkRouteResourceModel) error {
	route, err := r.client.PrivateNetworks.CreateRoute(ctx, data.PrivateNetworkID.ValueString(), data.SubnetID.ValueString(), verda.CreateRouteRequest{
		Destination: data.Destination.ValueString(),
		InstanceID:  data.InstanceID.ValueString(),
	})
	if err != nil {
		return err
	}
	setNetworkRouteModel(data, route)
	return nil
}

func (r *NetworkRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create private network route, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	routes, err := r.client.PrivateNetworks.GetRoutes(ctx, data.PrivateNetworkID.ValueString(), data.SubnetID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network routes, got error: %s", err))
		return
	}

	route := findRoute(routes, data.ID.ValueString(), data.Destination.ValueString())
	if route == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	setNetworkRouteModel(&data, route)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NetworkRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update private network route, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.PrivateNetworks.DeleteRoute(ctx, data.PrivateNetworkID.ValueString(), data.SubnetID.ValueString(), data.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete private network route, got error: %s", err))
	}
}

func (r *NetworkRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseImportID(req.ID, routeImportIDFormat)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_network_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subnet_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// findRoute returns the route with id or, if that id is gone (an upsert may
// have replaced it), the route for destination. It returns nil if neither
// matches.
func findRoute(routes []verda.Route, id, destination string) *verda.Route {
	for i := range routes {
		if routes[i].ID == id {
			return &routes[i]
		}
	}
	if destination == "" {
		return nil
	}
	for i := range routes {
		if routes[i].Destination == destination {
			return &routes[i]
		}
	}
	return nil
}

func setNetworkRouteModel(data *NetworkRouteResourceModel, route *verda.Route) {
	data.ID = types.StringValue(route.ID)
	data.Destination = types.StringValue(route.Destination)
	if route.SubnetID != "" {
		data.SubnetID = types.StringValue(route.SubnetID)
	}
	data.InstanceID = types.StringNull()
	if route.Instance != nil {
		data.InstanceID = types.StringValue(route.Instance.ID)
	}
	data.CreatedAt = types.StringValue(formatTimestamp(route.CreatedAt))
}
