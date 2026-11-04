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

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ datasource.DataSource = &PrivateNetworkDataSource{}
var _ datasource.DataSourceWithConfigValidators = &PrivateNetworkDataSource{}

func NewPrivateNetworkDataSource() datasource.DataSource {
	return &PrivateNetworkDataSource{}
}

type PrivateNetworkDataSource struct {
	client *verda.Client
}

type PrivateNetworkDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Default   types.Bool   `tfsdk:"default"`
	Mode      types.String `tfsdk:"mode"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *PrivateNetworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network"
}

func (d *PrivateNetworkDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Verda private network by ID, by name, " +
			"or as the project default. Set exactly one of `id`, `name` or `default`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Private network identifier (`pn_…`) to look up. Set from the result for other lookups.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the private network to look up. Set from the result for other lookups.",
			},
			"default": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` to look up the project's default private network",
			},
			"mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Subnet allocation mode: `auto` or `custom`",
			},
			"is_default": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is the project's default private network",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
			},
		},
	}
}

func (d *PrivateNetworkDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
			path.MatchRoot("default"),
		),
	}
}

func (d *PrivateNetworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PrivateNetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateNetworkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Default.IsNull() && !data.Default.ValueBool() {
		resp.Diagnostics.AddAttributeError(path.Root("default"), "Invalid default",
			"default can only be set to true; to look up another private network, set id or name instead")
		return
	}

	var network *verda.PrivateNetwork
	var err error
	if !data.ID.IsNull() {
		network, err = d.client.PrivateNetworks.GetByID(ctx, data.ID.ValueString())
	} else {
		var networks []verda.PrivateNetwork
		networks, err = d.client.PrivateNetworks.Get(ctx)
		if err == nil {
			network, err = findPrivateNetwork(networks, data.Name.ValueString())
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to find private network", err.Error())
		return
	}

	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)
	data.Mode = types.StringValue(network.Mode)
	data.IsDefault = types.BoolValue(network.IsDefault)
	data.CreatedAt = types.StringValue(formatTimestamp(network.CreatedAt))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
