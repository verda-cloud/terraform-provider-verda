package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

// Balance data source

type BalanceDataSource struct {
	client *verda.Client
}

type BalanceDataSourceModel struct {
	Amount   types.Float64 `tfsdk:"amount"`
	Currency types.String  `tfsdk:"currency"`
}

func NewBalanceDataSource() datasource.DataSource {
	return &BalanceDataSource{}
}

func (d *BalanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_balance"
}

func (d *BalanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the current account balance.",
		Attributes: map[string]schema.Attribute{
			"amount": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Balance amount.",
			},
			"currency": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Balance currency.",
			},
		},
	}
}

func (d *BalanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BalanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BalanceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	balance, err := d.client.Balance.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read balance, got error: %s", err))
		return
	}

	data.Amount = types.Float64Value(balance.Amount)
	data.Currency = types.StringValue(balance.Currency)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Locations data source

type LocationsDataSource struct {
	client *verda.Client
}

type LocationsDataSourceModel struct {
	Locations types.List `tfsdk:"locations"`
}

func NewLocationsDataSource() datasource.DataSource {
	return &LocationsDataSource{}
}

func (d *LocationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *LocationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches available locations.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Available locations.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"code": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"country_code": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *LocationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LocationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LocationsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	locations, err := d.client.Locations.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read locations, got error: %s", err))
		return
	}

	locationAttrTypes := map[string]attr.Type{
		"code":         types.StringType,
		"name":         types.StringType,
		"country_code": types.StringType,
	}

	var items []map[string]attr.Value
	for _, location := range locations {
		items = append(items, map[string]attr.Value{
			"code":         types.StringValue(location.Code),
			"name":         types.StringValue(location.Name),
			"country_code": types.StringValue(location.CountryCode),
		})
	}

	listValue, diags := objectListValue(locationAttrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Locations = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Images data source

type ImagesDataSource struct {
	client *verda.Client
}

type ImagesDataSourceModel struct {
	Images types.List `tfsdk:"images"`
}

func NewImagesDataSource() datasource.DataSource {
	return &ImagesDataSource{}
}

func (d *ImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func (d *ImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches available instance images.",
		Attributes: map[string]schema.Attribute{
			"images": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Instance images.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"image_type": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"is_default": schema.BoolAttribute{
							Computed: true,
						},
						"is_cluster": schema.BoolAttribute{
							Computed: true,
						},
						"details": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"category": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *ImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImagesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	images, err := d.client.Images.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read images, got error: %s", err))
		return
	}

	imageAttrTypes := map[string]attr.Type{
		"id":         types.StringType,
		"image_type": types.StringType,
		"name":       types.StringType,
		"is_default": types.BoolType,
		"is_cluster": types.BoolType,
		"details":    types.ListType{ElemType: types.StringType},
		"category":   types.StringType,
	}

	var items []map[string]attr.Value
	for _, image := range images {
		details, diags := stringListValue(ctx, image.Details)
		resp.Diagnostics.Append(diags...)
		items = append(items, map[string]attr.Value{
			"id":         types.StringValue(image.ID),
			"image_type": types.StringValue(image.ImageType),
			"name":       types.StringValue(image.Name),
			"is_default": types.BoolValue(image.IsDefault),
			"is_cluster": types.BoolValue(image.IsCluster),
			"details":    details,
			"category":   types.StringValue(image.Category),
		})
	}

	listValue, diags := objectListValue(imageAttrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Images = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Cluster images data source

type ClusterImagesDataSource struct {
	client *verda.Client
}

type ClusterImagesDataSourceModel struct {
	Images types.List `tfsdk:"images"`
}

func NewClusterImagesDataSource() datasource.DataSource {
	return &ClusterImagesDataSource{}
}

func (d *ClusterImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_images"
}

func (d *ClusterImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches available cluster images.",
		Attributes: map[string]schema.Attribute{
			"images": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster images.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *ClusterImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterImagesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	images, err := d.client.Images.GetClusterImages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster images, got error: %s", err))
		return
	}

	imageAttrTypes := map[string]attr.Type{
		"name":        types.StringType,
		"description": types.StringType,
	}

	var items []map[string]attr.Value
	for _, image := range images {
		items = append(items, map[string]attr.Value{
			"name":        types.StringValue(image.Name),
			"description": types.StringValue(image.Description),
		})
	}

	listValue, diags := objectListValue(imageAttrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Images = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Instance types data source

type InstanceTypesDataSource struct {
	client *verda.Client
}

type InstanceTypesDataSourceModel struct {
	Currency      types.String `tfsdk:"currency"`
	InstanceTypes types.List   `tfsdk:"instance_types"`
}

func NewInstanceTypesDataSource() datasource.DataSource {
	return &InstanceTypesDataSource{}
}

func (d *InstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_types"
}

func (d *InstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches available instance types.",
		Attributes: map[string]schema.Attribute{
			"currency": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Currency for pricing.",
			},
			"instance_types": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Instance types with pricing and specs.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"instance_type": schema.StringAttribute{
							Computed: true,
						},
						"model": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"cpu": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"number_of_cores": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"gpu": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"number_of_gpus": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"gpu_memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"storage": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
							},
						},
						"price_per_hour": schema.Float64Attribute{
							Computed: true,
						},
						"spot_price": schema.Float64Attribute{
							Computed: true,
						},
						"dynamic_price": schema.Float64Attribute{
							Computed: true,
						},
						"max_dynamic_price": schema.Float64Attribute{
							Computed: true,
						},
						"currency": schema.StringAttribute{
							Computed: true,
						},
						"manufacturer": schema.StringAttribute{
							Computed: true,
						},
						"best_for": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *InstanceTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceTypesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	currency := ""
	if !data.Currency.IsNull() && !data.Currency.IsUnknown() {
		currency = data.Currency.ValueString()
	}

	instanceTypes, err := d.client.InstanceTypes.Get(ctx, currency)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance types, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":            types.StringType,
		"instance_type": types.StringType,
		"model":         types.StringType,
		"name":          types.StringType,
		"cpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		}},
		"gpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":    types.StringType,
			"number_of_gpus": types.Int64Type,
		}},
		"gpu_memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"storage": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description": types.StringType,
		}},
		"price_per_hour":    types.Float64Type,
		"spot_price":        types.Float64Type,
		"dynamic_price":     types.Float64Type,
		"max_dynamic_price": types.Float64Type,
		"currency":          types.StringType,
		"manufacturer":      types.StringType,
		"best_for":          types.ListType{ElemType: types.StringType},
		"description":       types.StringType,
	}

	var items []map[string]attr.Value
	for _, instanceType := range instanceTypes {
		cpuObj, diags := cpuObjectValue(instanceType.CPU)
		resp.Diagnostics.Append(diags...)
		gpuObj, diags := gpuObjectValue(instanceType.GPU)
		resp.Diagnostics.Append(diags...)
		gpuMemoryObj, diags := memoryObjectValue(instanceType.GPUMemory)
		resp.Diagnostics.Append(diags...)
		memoryObj, diags := memoryObjectValue(instanceType.Memory)
		resp.Diagnostics.Append(diags...)
		storageObj, diags := storageObjectValue(instanceType.Storage)
		resp.Diagnostics.Append(diags...)
		bestFor, diags := stringListValue(ctx, instanceType.BestFor)
		resp.Diagnostics.Append(diags...)

		items = append(items, map[string]attr.Value{
			"id":                types.StringValue(instanceType.ID),
			"instance_type":     types.StringValue(instanceType.InstanceType),
			"model":             types.StringValue(instanceType.Model),
			"name":              types.StringValue(instanceType.Name),
			"cpu":               cpuObj,
			"gpu":               gpuObj,
			"gpu_memory":        gpuMemoryObj,
			"memory":            memoryObj,
			"storage":           storageObj,
			"price_per_hour":    types.Float64Value(instanceType.PricePerHour.Float64()),
			"spot_price":        types.Float64Value(instanceType.SpotPrice.Float64()),
			"dynamic_price":     types.Float64Value(instanceType.DynamicPrice.Float64()),
			"max_dynamic_price": types.Float64Value(instanceType.MaxDynamicPrice.Float64()),
			"currency":          types.StringValue(instanceType.Currency),
			"manufacturer":      types.StringValue(instanceType.Manufacturer),
			"best_for":          bestFor,
			"description":       types.StringValue(instanceType.Description),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.InstanceTypes = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Instance type price history data source

type InstanceTypePriceHistoryDataSource struct {
	client *verda.Client
}

type InstanceTypePriceHistoryDataSourceModel struct {
	NumOfMonths  types.Int64  `tfsdk:"num_of_months"`
	Currency     types.String `tfsdk:"currency"`
	PriceHistory types.Map    `tfsdk:"price_history"`
}

func NewInstanceTypePriceHistoryDataSource() datasource.DataSource {
	return &InstanceTypePriceHistoryDataSource{}
}

func (d *InstanceTypePriceHistoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_type_price_history"
}

func (d *InstanceTypePriceHistoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	recordAttrTypes := map[string]attr.Type{
		"date":                   types.StringType,
		"fixed_price_per_hour":   types.Float64Type,
		"dynamic_price_per_hour": types.Float64Type,
		"currency":               types.StringType,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches instance type price history.",
		Attributes: map[string]schema.Attribute{
			"num_of_months": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Number of months to query.",
			},
			"currency": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Currency for pricing.",
			},
			"price_history": schema.MapAttribute{
				Computed:            true,
				MarkdownDescription: "Map of instance type to price history records.",
				ElementType: types.ListType{
					ElemType: types.ObjectType{AttrTypes: recordAttrTypes},
				},
			},
		},
	}
}

func (d *InstanceTypePriceHistoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceTypePriceHistoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceTypePriceHistoryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	numOfMonths := 0
	if !data.NumOfMonths.IsNull() && !data.NumOfMonths.IsUnknown() {
		numOfMonths = int(data.NumOfMonths.ValueInt64())
	}

	currency := ""
	if !data.Currency.IsNull() && !data.Currency.IsUnknown() {
		currency = data.Currency.ValueString()
	}

	priceHistory, err := d.client.InstanceTypes.GetPriceHistory(ctx, numOfMonths, currency)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read price history, got error: %s", err))
		return
	}

	recordAttrTypes := map[string]attr.Type{
		"date":                   types.StringType,
		"fixed_price_per_hour":   types.Float64Type,
		"dynamic_price_per_hour": types.Float64Type,
		"currency":               types.StringType,
	}

	mapValues := map[string]attr.Value{}
	for instanceType, records := range priceHistory {
		var recordItems []map[string]attr.Value
		for _, record := range records {
			recordItems = append(recordItems, map[string]attr.Value{
				"date":                   types.StringValue(record.Date),
				"fixed_price_per_hour":   types.Float64Value(record.FixedPricePerHour.Float64()),
				"dynamic_price_per_hour": types.Float64Value(record.DynamicPricePerHour.Float64()),
				"currency":               types.StringValue(record.Currency),
			})
		}

		listValue, diags := objectListValue(recordAttrTypes, recordItems)
		resp.Diagnostics.Append(diags...)
		mapValues[instanceType] = listValue
	}

	mapValue, diags := types.MapValue(
		types.ListType{ElemType: types.ObjectType{AttrTypes: recordAttrTypes}},
		mapValues,
	)
	resp.Diagnostics.Append(diags...)
	data.PriceHistory = mapValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Instance availability data source (list)

type InstanceAvailabilityDataSource struct {
	client *verda.Client
}

type InstanceAvailabilityDataSourceModel struct {
	IsSpot         types.Bool   `tfsdk:"is_spot"`
	LocationCode   types.String `tfsdk:"location_code"`
	Availabilities types.List   `tfsdk:"availabilities"`
}

func NewInstanceAvailabilityDataSource() datasource.DataSource {
	return &InstanceAvailabilityDataSource{}
}

func (d *InstanceAvailabilityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_availability"
}

func (d *InstanceAvailabilityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches instance type availability by location.",
		Attributes: map[string]schema.Attribute{
			"is_spot": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Check spot availability.",
			},
			"location_code": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Location code filter.",
			},
			"availabilities": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Availability by location.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"location_code": schema.StringAttribute{Computed: true},
						"availabilities": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *InstanceAvailabilityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceAvailabilityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceAvailabilityDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isSpot := false
	if !data.IsSpot.IsNull() && !data.IsSpot.IsUnknown() {
		isSpot = data.IsSpot.ValueBool()
	}

	locationCode := ""
	if !data.LocationCode.IsNull() && !data.LocationCode.IsUnknown() {
		locationCode = data.LocationCode.ValueString()
	}

	availabilities, err := d.client.InstanceAvailability.GetAllAvailabilities(ctx, isSpot, locationCode)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance availability, got error: %s", err))
		return
	}

	availabilityAttrTypes := map[string]attr.Type{
		"location_code": types.StringType,
		"availabilities": types.ListType{
			ElemType: types.StringType,
		},
	}

	var items []map[string]attr.Value
	for _, availability := range availabilities {
		availableList, diags := stringListValue(ctx, availability.Availabilities)
		resp.Diagnostics.Append(diags...)
		items = append(items, map[string]attr.Value{
			"location_code":  types.StringValue(availability.LocationCode),
			"availabilities": availableList,
		})
	}

	listValue, diags := objectListValue(availabilityAttrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Availabilities = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Instance type availability data source (single check)

type InstanceTypeAvailabilityDataSource struct {
	client *verda.Client
}

type InstanceTypeAvailabilityDataSourceModel struct {
	InstanceType types.String `tfsdk:"instance_type"`
	IsSpot       types.Bool   `tfsdk:"is_spot"`
	LocationCode types.String `tfsdk:"location_code"`
	Available    types.Bool   `tfsdk:"available"`
}

func NewInstanceTypeAvailabilityDataSource() datasource.DataSource {
	return &InstanceTypeAvailabilityDataSource{}
}

func (d *InstanceTypeAvailabilityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_type_availability"
}

func (d *InstanceTypeAvailabilityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks availability for a specific instance type.",
		Attributes: map[string]schema.Attribute{
			"instance_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Instance type to check.",
			},
			"is_spot": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Check spot availability.",
			},
			"location_code": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Location code filter.",
			},
			"available": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Availability result.",
			},
		},
	}
}

func (d *InstanceTypeAvailabilityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceTypeAvailabilityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceTypeAvailabilityDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isSpot := false
	if !data.IsSpot.IsNull() && !data.IsSpot.IsUnknown() {
		isSpot = data.IsSpot.ValueBool()
	}

	locationCode := ""
	if !data.LocationCode.IsNull() && !data.LocationCode.IsUnknown() {
		locationCode = data.LocationCode.ValueString()
	}

	available, err := d.client.InstanceAvailability.GetInstanceTypeAvailability(
		ctx,
		data.InstanceType.ValueString(),
		isSpot,
		locationCode,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check instance type availability, got error: %s", err))
		return
	}

	data.Available = types.BoolValue(available)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Cluster types data source

type ClusterTypesDataSource struct {
	client *verda.Client
}

type ClusterTypesDataSourceModel struct {
	Currency     types.String `tfsdk:"currency"`
	ClusterTypes types.List   `tfsdk:"cluster_types"`
}

func NewClusterTypesDataSource() datasource.DataSource {
	return &ClusterTypesDataSource{}
}

func (d *ClusterTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_types"
}

func (d *ClusterTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches available cluster types.",
		Attributes: map[string]schema.Attribute{
			"currency": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Currency for pricing.",
			},
			"cluster_types": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster type specifications.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cluster_type": schema.StringAttribute{Computed: true},
						"description":  schema.StringAttribute{Computed: true},
						"price_per_hour": schema.Float64Attribute{
							Computed: true,
						},
						"cpu": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"number_of_cores": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"gpu": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"number_of_gpus": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"gpu_memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"storage": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
							},
						},
						"manufacturer": schema.StringAttribute{Computed: true},
						"available": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *ClusterTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterTypesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	currency := ""
	if !data.Currency.IsNull() && !data.Currency.IsUnknown() {
		currency = data.Currency.ValueString()
	}

	clusterTypes, err := d.client.Clusters.GetClusterTypes(ctx, currency)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster types, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"cluster_type":   types.StringType,
		"description":    types.StringType,
		"price_per_hour": types.Float64Type,
		"cpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		}},
		"gpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":    types.StringType,
			"number_of_gpus": types.Int64Type,
		}},
		"gpu_memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"storage": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description": types.StringType,
		}},
		"manufacturer": types.StringType,
		"available":    types.BoolType,
	}

	var items []map[string]attr.Value
	for _, clusterType := range clusterTypes {
		cpuObj, diags := cpuObjectValue(clusterType.CPU)
		resp.Diagnostics.Append(diags...)
		gpuObj, diags := gpuObjectValue(clusterType.GPU)
		resp.Diagnostics.Append(diags...)
		gpuMemoryObj, diags := memoryObjectValue(clusterType.GPUMemory)
		resp.Diagnostics.Append(diags...)
		memoryObj, diags := memoryObjectValue(clusterType.Memory)
		resp.Diagnostics.Append(diags...)
		storageObj, diags := storageObjectValue(clusterType.Storage)
		resp.Diagnostics.Append(diags...)

		items = append(items, map[string]attr.Value{
			"cluster_type":   types.StringValue(clusterType.ClusterType),
			"description":    types.StringValue(clusterType.Description),
			"price_per_hour": types.Float64Value(clusterType.PricePerHour.Float64()),
			"cpu":            cpuObj,
			"gpu":            gpuObj,
			"gpu_memory":     gpuMemoryObj,
			"memory":         memoryObj,
			"storage":        storageObj,
			"manufacturer":   types.StringValue(clusterType.Manufacturer),
			"available":      types.BoolValue(clusterType.Available),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.ClusterTypes = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Cluster availability data source (list)

type ClusterAvailabilityDataSource struct {
	client *verda.Client
}

type ClusterAvailabilityDataSourceModel struct {
	LocationCode   types.String `tfsdk:"location_code"`
	Availabilities types.List   `tfsdk:"availabilities"`
}

type clusterAvailabilityResponse struct {
	LocationCode   string   `json:"location_code"`
	Availabilities []string `json:"availabilities"`
}

func NewClusterAvailabilityDataSource() datasource.DataSource {
	return &ClusterAvailabilityDataSource{}
}

func (d *ClusterAvailabilityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_availability"
}

func (d *ClusterAvailabilityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches cluster type availability by location.",
		Attributes: map[string]schema.Attribute{
			"location_code": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Location code filter.",
			},
			"availabilities": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Availability by location.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"location_code": schema.StringAttribute{Computed: true},
						"availabilities": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ClusterAvailabilityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterAvailabilityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterAvailabilityDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := "/cluster-availability"
	if !data.LocationCode.IsNull() && !data.LocationCode.IsUnknown() && data.LocationCode.ValueString() != "" {
		params := url.Values{}
		params.Set("location_code", data.LocationCode.ValueString())
		path += "?" + params.Encode()
	}

	var response []clusterAvailabilityResponse
	if err := doVerdaRequest(ctx, d.client, "GET", path, nil, &response); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster availability, got error: %s", err))
		return
	}

	availabilityAttrTypes := map[string]attr.Type{
		"location_code": types.StringType,
		"availabilities": types.ListType{
			ElemType: types.StringType,
		},
	}

	var items []map[string]attr.Value
	for _, availability := range response {
		availableList, diags := stringListValue(ctx, availability.Availabilities)
		resp.Diagnostics.Append(diags...)
		items = append(items, map[string]attr.Value{
			"location_code":  types.StringValue(availability.LocationCode),
			"availabilities": availableList,
		})
	}

	listValue, diags := objectListValue(availabilityAttrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Availabilities = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Cluster type availability data source (single check)

type ClusterTypeAvailabilityDataSource struct {
	client *verda.Client
}

type ClusterTypeAvailabilityDataSourceModel struct {
	ClusterType  types.String `tfsdk:"cluster_type"`
	LocationCode types.String `tfsdk:"location_code"`
	Available    types.Bool   `tfsdk:"available"`
}

func NewClusterTypeAvailabilityDataSource() datasource.DataSource {
	return &ClusterTypeAvailabilityDataSource{}
}

func (d *ClusterTypeAvailabilityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_type_availability"
}

func (d *ClusterTypeAvailabilityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks availability for a specific cluster type.",
		Attributes: map[string]schema.Attribute{
			"cluster_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Cluster type to check.",
			},
			"location_code": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Location code filter.",
			},
			"available": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Availability result.",
			},
		},
	}
}

func (d *ClusterTypeAvailabilityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterTypeAvailabilityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterTypeAvailabilityDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := fmt.Sprintf("/cluster-availability/%s", data.ClusterType.ValueString())
	if !data.LocationCode.IsNull() && !data.LocationCode.IsUnknown() && data.LocationCode.ValueString() != "" {
		params := url.Values{}
		params.Set("location_code", data.LocationCode.ValueString())
		path += "?" + params.Encode()
	}

	var response bool
	if err := doVerdaRequest(ctx, d.client, "GET", path, nil, &response); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check cluster type availability, got error: %s", err))
		return
	}

	data.Available = types.BoolValue(response)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Container types data source

type ContainerTypesDataSource struct {
	client *verda.Client
}

type ContainerTypesDataSourceModel struct {
	Currency       types.String `tfsdk:"currency"`
	ContainerTypes types.List   `tfsdk:"container_types"`
}

func NewContainerTypesDataSource() datasource.DataSource {
	return &ContainerTypesDataSource{}
}

func (d *ContainerTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_types"
}

func (d *ContainerTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches available container types.",
		Attributes: map[string]schema.Attribute{
			"currency": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Currency for pricing.",
			},
			"container_types": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Container types with pricing and specs.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"model":         schema.StringAttribute{Computed: true},
						"name":          schema.StringAttribute{Computed: true},
						"instance_type": schema.StringAttribute{Computed: true},
						"cpu": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"number_of_cores": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"gpu": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"number_of_gpus": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"gpu_memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"serverless_price": schema.Float64Attribute{
							Computed: true,
						},
						"serverless_spot_price": schema.Float64Attribute{
							Computed: true,
						},
						"currency":     schema.StringAttribute{Computed: true},
						"manufacturer": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ContainerTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContainerTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerTypesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	currency := ""
	if !data.Currency.IsNull() && !data.Currency.IsUnknown() {
		currency = data.Currency.ValueString()
	}

	containerTypes, err := d.client.ContainerTypes.Get(ctx, currency)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container types, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":            types.StringType,
		"model":         types.StringType,
		"name":          types.StringType,
		"instance_type": types.StringType,
		"cpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		}},
		"gpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":    types.StringType,
			"number_of_gpus": types.Int64Type,
		}},
		"gpu_memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"serverless_price":      types.Float64Type,
		"serverless_spot_price": types.Float64Type,
		"currency":              types.StringType,
		"manufacturer":          types.StringType,
	}

	var items []map[string]attr.Value
	for _, containerType := range containerTypes {
		cpuObj, diags := cpuObjectValue(containerType.CPU)
		resp.Diagnostics.Append(diags...)
		gpuObj, diags := gpuObjectValue(containerType.GPU)
		resp.Diagnostics.Append(diags...)
		gpuMemoryObj, diags := memoryObjectValue(containerType.GPUMemory)
		resp.Diagnostics.Append(diags...)
		memoryObj, diags := memoryObjectValue(containerType.Memory)
		resp.Diagnostics.Append(diags...)

		items = append(items, map[string]attr.Value{
			"id":                    types.StringValue(containerType.ID),
			"model":                 types.StringValue(containerType.Model),
			"name":                  types.StringValue(containerType.Name),
			"instance_type":         types.StringValue(containerType.InstanceType),
			"cpu":                   cpuObj,
			"gpu":                   gpuObj,
			"gpu_memory":            gpuMemoryObj,
			"memory":                memoryObj,
			"serverless_price":      types.Float64Value(containerType.ServerlessPrice.Float64()),
			"serverless_spot_price": types.Float64Value(containerType.ServerlessSpotPrice.Float64()),
			"currency":              types.StringValue(containerType.Currency),
			"manufacturer":          types.StringValue(containerType.Manufacturer),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.ContainerTypes = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Volume types data source

type VolumeTypesDataSource struct {
	client *verda.Client
}

type VolumeTypesDataSourceModel struct {
	VolumeTypes types.List `tfsdk:"volume_types"`
}

func NewVolumeTypesDataSource() datasource.DataSource {
	return &VolumeTypesDataSource{}
}

func (d *VolumeTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_types"
}

func (d *VolumeTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches available volume types.",
		Attributes: map[string]schema.Attribute{
			"volume_types": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Volume type pricing details.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{Computed: true},
						"price": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"monthly_per_gb": schema.Float64Attribute{Computed: true},
								"currency":       schema.StringAttribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

func (d *VolumeTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VolumeTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VolumeTypesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volumeTypes, err := d.client.VolumeTypes.GetAllVolumeTypes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume types, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"type": types.StringType,
		"price": types.ObjectType{AttrTypes: map[string]attr.Type{
			"monthly_per_gb": types.Float64Type,
			"currency":       types.StringType,
		}},
	}

	var items []map[string]attr.Value
	for _, volumeType := range volumeTypes {
		priceObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"monthly_per_gb": types.Float64Type,
				"currency":       types.StringType,
			},
			map[string]attr.Value{
				"monthly_per_gb": types.Float64Value(volumeType.Price.MonthlyPerGB),
				"currency":       types.StringValue(volumeType.Price.Currency),
			},
		)
		resp.Diagnostics.Append(diags...)

		items = append(items, map[string]attr.Value{
			"type":  types.StringValue(volumeType.Type),
			"price": priceObj,
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.VolumeTypes = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Serverless compute resources data source

type ServerlessComputeResourcesDataSource struct {
	client *verda.Client
}

type ServerlessComputeResourcesDataSourceModel struct {
	Resources types.List `tfsdk:"resources"`
}

type serverlessComputeResource struct {
	Name        string `json:"name"`
	Size        int    `json:"size"`
	IsAvailable bool   `json:"is_available"`
}

func NewServerlessComputeResourcesDataSource() datasource.DataSource {
	return &ServerlessComputeResourcesDataSource{}
}

func (d *ServerlessComputeResourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_compute_resources"
}

func (d *ServerlessComputeResourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches serverless compute resources and availability.",
		Attributes: map[string]schema.Attribute{
			"resources": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Serverless compute resources.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"size": schema.Int64Attribute{Computed: true},
						"is_available": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *ServerlessComputeResourcesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServerlessComputeResourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerlessComputeResourcesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var response []serverlessComputeResource
	if err := doVerdaRequest(ctx, d.client, "GET", "/serverless-compute-resources", nil, &response); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read serverless compute resources, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"name":         types.StringType,
		"size":         types.Int64Type,
		"is_available": types.BoolType,
	}

	var items []map[string]attr.Value
	for _, resource := range response {
		items = append(items, map[string]attr.Value{
			"name":         types.StringValue(resource.Name),
			"size":         types.Int64Value(int64(resource.Size)),
			"is_available": types.BoolValue(resource.IsAvailable),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Resources = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Long-term periods data sources

type LongTermPeriodsDataSource struct {
	client *verda.Client
}

type LongTermPeriodsDataSourceModel struct {
	Periods types.List `tfsdk:"periods"`
}

func NewLongTermPeriodsDataSource() datasource.DataSource {
	return &LongTermPeriodsDataSource{}
}

func (d *LongTermPeriodsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_long_term_periods"
}

func (d *LongTermPeriodsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches long-term rental periods.",
		Attributes: map[string]schema.Attribute{
			"periods": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Long-term period options.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"code": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"is_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"unit_name": schema.StringAttribute{Computed: true},
						"unit_value": schema.Int64Attribute{
							Computed: true,
						},
						"discount_percentage": schema.Float64Attribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *LongTermPeriodsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LongTermPeriodsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LongTermPeriodsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	periods, err := d.client.LongTerm.GetPeriods(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read long-term periods, got error: %s", err))
		return
	}

	data.Periods = flattenLongTermPeriods(ctx, periods, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type LongTermInstancePeriodsDataSource struct {
	client *verda.Client
}

type LongTermInstancePeriodsDataSourceModel struct {
	Periods types.List `tfsdk:"periods"`
}

func NewLongTermInstancePeriodsDataSource() datasource.DataSource {
	return &LongTermInstancePeriodsDataSource{}
}

func (d *LongTermInstancePeriodsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_long_term_instance_periods"
}

func (d *LongTermInstancePeriodsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches long-term rental periods for instances.",
		Attributes: map[string]schema.Attribute{
			"periods": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Instance long-term period options.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"code": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"is_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"unit_name": schema.StringAttribute{Computed: true},
						"unit_value": schema.Int64Attribute{
							Computed: true,
						},
						"discount_percentage": schema.Float64Attribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *LongTermInstancePeriodsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LongTermInstancePeriodsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LongTermInstancePeriodsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	periods, err := d.client.LongTerm.GetInstancePeriods(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance periods, got error: %s", err))
		return
	}

	data.Periods = flattenLongTermPeriods(ctx, periods, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type LongTermClusterPeriodsDataSource struct {
	client *verda.Client
}

type LongTermClusterPeriodsDataSourceModel struct {
	Periods types.List `tfsdk:"periods"`
}

func NewLongTermClusterPeriodsDataSource() datasource.DataSource {
	return &LongTermClusterPeriodsDataSource{}
}

func (d *LongTermClusterPeriodsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_long_term_cluster_periods"
}

func (d *LongTermClusterPeriodsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches long-term rental periods for clusters.",
		Attributes: map[string]schema.Attribute{
			"periods": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster long-term period options.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"code": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"is_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"unit_name": schema.StringAttribute{Computed: true},
						"unit_value": schema.Int64Attribute{
							Computed: true,
						},
						"discount_percentage": schema.Float64Attribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *LongTermClusterPeriodsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LongTermClusterPeriodsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LongTermClusterPeriodsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	periods, err := d.client.LongTerm.GetClusterPeriods(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster periods, got error: %s", err))
		return
	}

	data.Periods = flattenLongTermPeriods(ctx, periods, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func flattenLongTermPeriods(ctx context.Context, periods []verda.LongTermPeriod, diagnostics *diag.Diagnostics) types.List {
	attrTypes := map[string]attr.Type{
		"code":                types.StringType,
		"name":                types.StringType,
		"is_enabled":          types.BoolType,
		"unit_name":           types.StringType,
		"unit_value":          types.Int64Type,
		"discount_percentage": types.Float64Type,
	}

	var items []map[string]attr.Value
	for _, period := range periods {
		items = append(items, map[string]attr.Value{
			"code":                types.StringValue(period.Code),
			"name":                types.StringValue(period.Name),
			"is_enabled":          types.BoolValue(period.IsEnabled),
			"unit_name":           types.StringValue(period.UnitName),
			"unit_value":          types.Int64Value(int64(period.UnitValue)),
			"discount_percentage": types.Float64Value(period.DiscountPercentage),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	diagnostics.Append(diags...)
	return listValue
}
