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

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ datasource.DataSource = &SubnetDataSource{}
var _ datasource.DataSourceWithConfigValidators = &SubnetDataSource{}

func NewSubnetDataSource() datasource.DataSource {
	return &SubnetDataSource{}
}

type SubnetDataSource struct {
	client *verda.Client
}

type SubnetDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	PrivateNetworkID       types.String `tfsdk:"private_network_id"`
	Location               types.String `tfsdk:"location"`
	Name                   types.String `tfsdk:"name"`
	Default                types.Bool   `tfsdk:"default"`
	CIDR                   types.String `tfsdk:"cidr"`
	IsDefault              types.Bool   `tfsdk:"is_default"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UsedIPCount            types.Int64  `tfsdk:"used_ip_count"`
	TotalIPCount           types.Int64  `tfsdk:"total_ip_count"`
	DefaultRouteInstanceID types.String `tfsdk:"default_route_instance_id"`
}

func (d *SubnetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (d *SubnetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a subnet of a Verda private network by ID, by name, or as the private network's default " +
			"subnet in a location. Set `private_network_id` and exactly one of `id`, `name` or `default` " +
			"(`default` also needs `location`).",

		Attributes: map[string]schema.Attribute{
			"private_network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the private network to search",
			},
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Subnet identifier (`sn_…`) to look up. Set from the result for other lookups.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the subnet to look up. Set from the result for other lookups.",
			},
			"location": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Location code, e.g. `FIN-03`. Required with `default`; " +
					"with `id` or `name` it narrows the search.",
			},
			"default": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` to look up the private network's default subnet in `location`",
				Validators: []validator.Bool{
					boolvalidator.AlsoRequires(path.MatchRoot("location")),
				},
			},
			"cidr": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IPv4 range of the subnet",
			},
			"is_default": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is the private network's default subnet in its location",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
			},
			"used_ip_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of addresses in use",
			},
			"total_ip_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of usable addresses",
			},
			"default_route_instance_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the instance the subnet's default route (`0.0.0.0/0`) points at, if any",
			},
		},
	}
}

func (d *SubnetDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
			path.MatchRoot("default"),
		),
	}
}

func (d *SubnetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*verda.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *verda.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *SubnetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SubnetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Default.IsNull() && !data.Default.ValueBool() {
		resp.Diagnostics.AddAttributeError(path.Root("default"), "Invalid default",
			"default can only be set to true; to look up another subnet, set id or name instead")
		return
	}

	privateNetworkID := data.PrivateNetworkID.ValueString()
	location := data.Location.ValueString()

	var subnet *verda.Subnet
	var err error
	if !data.ID.IsNull() {
		subnet, err = d.client.PrivateNetworks.GetSubnetByID(ctx, privateNetworkID, data.ID.ValueString())
		if err == nil && location != "" && subnet.Location.Code != location {
			err = fmt.Errorf("subnet %s is in %s, not %s", subnet.ID, subnet.Location.Code, location)
		}
	} else {
		var subnets []verda.Subnet
		subnets, err = d.client.PrivateNetworks.GetSubnets(ctx, privateNetworkID)
		if err == nil {
			subnet, err = findSubnet(subnets, data.Name.ValueString(), location)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to find private network subnet", err.Error())
		return
	}

	data.ID = types.StringValue(subnet.ID)
	data.Location = types.StringValue(subnet.Location.Code)
	data.Name = types.StringValue(subnet.Name)
	data.CIDR = types.StringValue(subnet.CIDR)
	data.IsDefault = types.BoolValue(subnet.IsDefault)
	data.CreatedAt = types.StringValue(formatTimestamp(subnet.CreatedAt))
	data.UsedIPCount = types.Int64Value(int64(subnet.UsedIPCount))
	data.TotalIPCount = types.Int64Value(int64(subnet.TotalIPCount))
	data.DefaultRouteInstanceID = types.StringNull()
	if subnet.DefaultRouteInstance != nil {
		data.DefaultRouteInstanceID = types.StringValue(subnet.DefaultRouteInstance.ID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
