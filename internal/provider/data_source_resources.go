package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

// Instances data source

type InstancesDataSource struct {
	client *verda.Client
}

type InstancesDataSourceModel struct {
	Status    types.String `tfsdk:"status"`
	Instances types.List   `tfsdk:"instances"`
}

func NewInstancesDataSource() datasource.DataSource {
	return &InstancesDataSource{}
}

func (d *InstancesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instances"
}

func (d *InstancesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches instances, optionally filtered by status.",
		Attributes: map[string]schema.Attribute{
			"status": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter instances by status.",
			},
			"instances": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Instances returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"instance_type": schema.StringAttribute{Computed: true},
						"image":         schema.StringAttribute{Computed: true},
						"hostname":      schema.StringAttribute{Computed: true},
						"description":   schema.StringAttribute{Computed: true},
						"price_per_hour": schema.Float64Attribute{
							Computed: true,
						},
						"ip":     schema.StringAttribute{Computed: true},
						"status": schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"ssh_key_ids": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"location": schema.StringAttribute{Computed: true},
						"is_spot": schema.BoolAttribute{
							Computed: true,
						},
						"os_name": schema.StringAttribute{Computed: true},
						"startup_script_id": schema.StringAttribute{
							Computed: true,
						},
						"os_volume_id": schema.StringAttribute{
							Computed: true,
						},
						"contract": schema.StringAttribute{Computed: true},
						"pricing":  schema.StringAttribute{Computed: true},
						"jupyter_token": schema.StringAttribute{
							Computed:  true,
							Sensitive: true,
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
						"memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
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
						"storage": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

func (d *InstancesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstancesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status := ""
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		status = data.Status.ValueString()
	}

	instances, err := d.client.Instances.Get(ctx, status)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instances, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":                types.StringType,
		"instance_type":     types.StringType,
		"image":             types.StringType,
		"hostname":          types.StringType,
		"description":       types.StringType,
		"price_per_hour":    types.Float64Type,
		"ip":                types.StringType,
		"status":            types.StringType,
		"created_at":        types.StringType,
		"ssh_key_ids":       types.ListType{ElemType: types.StringType},
		"location":          types.StringType,
		"is_spot":           types.BoolType,
		"os_name":           types.StringType,
		"startup_script_id": types.StringType,
		"os_volume_id":      types.StringType,
		"contract":          types.StringType,
		"pricing":           types.StringType,
		"jupyter_token":     types.StringType,
		"cpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		}},
		"gpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":    types.StringType,
			"number_of_gpus": types.Int64Type,
		}},
		"memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"gpu_memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"storage": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description": types.StringType,
		}},
	}

	var items []map[string]attr.Value
	for _, instance := range instances {
		cpuObj, diags := cpuObjectValue(instance.CPU)
		resp.Diagnostics.Append(diags...)
		gpuObj, diags := gpuObjectValue(instance.GPU)
		resp.Diagnostics.Append(diags...)
		memoryObj, diags := memoryObjectValue(instance.Memory)
		resp.Diagnostics.Append(diags...)
		gpuMemoryObj, diags := memoryObjectValue(instance.GPUMemory)
		resp.Diagnostics.Append(diags...)
		storageObj, diags := storageObjectValue(instance.Storage)
		resp.Diagnostics.Append(diags...)
		sshKeys, diags := stringListValue(ctx, instance.SSHKeyIDs)
		resp.Diagnostics.Append(diags...)

		items = append(items, map[string]attr.Value{
			"id":                types.StringValue(instance.ID),
			"instance_type":     types.StringValue(instance.InstanceType),
			"image":             types.StringValue(instance.Image),
			"hostname":          types.StringValue(instance.Hostname),
			"description":       types.StringValue(instance.Description),
			"price_per_hour":    types.Float64Value(instance.PricePerHour.Float64()),
			"ip":                stringPointerValueOrNull(instance.IP),
			"status":            types.StringValue(instance.Status),
			"created_at":        types.StringValue(instance.CreatedAt.Format("2006-01-02T15:04:05Z")),
			"ssh_key_ids":       sshKeys,
			"location":          types.StringValue(instance.Location),
			"is_spot":           types.BoolValue(instance.IsSpot),
			"os_name":           types.StringValue(instance.OSName),
			"startup_script_id": stringPointerValueOrNull(instance.StartupScriptID),
			"os_volume_id":      stringPointerValueOrNull(instance.OSVolumeID),
			"contract":          types.StringValue(instance.Contract),
			"pricing":           types.StringValue(instance.Pricing),
			"jupyter_token":     stringPointerValueOrNull(instance.JupyterToken),
			"cpu":               cpuObj,
			"gpu":               gpuObj,
			"memory":            memoryObj,
			"gpu_memory":        gpuMemoryObj,
			"storage":           storageObj,
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Instances = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Volumes data source

type VolumesDataSource struct {
	client *verda.Client
}

type VolumesDataSourceModel struct {
	Volumes types.List `tfsdk:"volumes"`
}

func NewVolumesDataSource() datasource.DataSource {
	return &VolumesDataSource{}
}

func (d *VolumesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volumes"
}

func (d *VolumesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches volumes.",
		Attributes: map[string]schema.Attribute{
			"volumes": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Volumes returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"size": schema.Int64Attribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"status": schema.StringAttribute{
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"instance_id": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *VolumesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VolumesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VolumesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volumes, err := d.client.Volumes.GetVolumes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volumes, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"size":        types.Int64Type,
		"type":        types.StringType,
		"status":      types.StringType,
		"created_at":  types.StringType,
		"instance_id": types.StringType,
	}

	var items []map[string]attr.Value
	for _, volume := range volumes {
		items = append(items, map[string]attr.Value{
			"id":          types.StringValue(volume.ID),
			"name":        types.StringValue(volume.Name),
			"size":        types.Int64Value(int64(volume.Size)),
			"type":        types.StringValue(volume.Type),
			"status":      types.StringValue(volume.Status),
			"created_at":  types.StringValue(volume.CreatedAt.Format("2006-01-02T15:04:05Z")),
			"instance_id": stringPointerValueOrNull(volume.InstanceID),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Volumes = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Clusters data source

type ClustersDataSource struct {
	client *verda.Client
}

type ClustersDataSourceModel struct {
	Clusters types.List `tfsdk:"clusters"`
}

func NewClustersDataSource() datasource.DataSource {
	return &ClustersDataSource{}
}

func (d *ClustersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clusters"
}

func (d *ClustersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches clusters.",
		Attributes: map[string]schema.Attribute{
			"clusters": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Clusters returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
						"cluster_type": schema.StringAttribute{Computed: true},
						"image":        schema.StringAttribute{Computed: true},
						"price_per_hour": schema.Float64Attribute{
							Computed: true,
						},
						"hostname":    schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"ip":          schema.StringAttribute{Computed: true},
						"status":      schema.StringAttribute{Computed: true},
						"created_at":  schema.StringAttribute{Computed: true},
						"ssh_key_ids": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
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
						"memory": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
								"size_in_gigabytes": schema.Int64Attribute{
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
						"storage": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"description": schema.StringAttribute{Computed: true},
							},
						},
						"location": schema.StringAttribute{Computed: true},
						"os_name":  schema.StringAttribute{Computed: true},
						"startup_script_id": schema.StringAttribute{
							Computed: true,
						},
						"contract": schema.StringAttribute{Computed: true},
						"pricing":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ClustersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClustersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusters, err := d.client.Clusters.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read clusters, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":             types.StringType,
		"cluster_type":   types.StringType,
		"image":          types.StringType,
		"price_per_hour": types.Float64Type,
		"hostname":       types.StringType,
		"description":    types.StringType,
		"ip":             types.StringType,
		"status":         types.StringType,
		"created_at":     types.StringType,
		"ssh_key_ids":    types.ListType{ElemType: types.StringType},
		"cpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		}},
		"gpu": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":    types.StringType,
			"number_of_gpus": types.Int64Type,
		}},
		"memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"gpu_memory": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		}},
		"storage": types.ObjectType{AttrTypes: map[string]attr.Type{
			"description": types.StringType,
		}},
		"location":          types.StringType,
		"os_name":           types.StringType,
		"startup_script_id": types.StringType,
		"contract":          types.StringType,
		"pricing":           types.StringType,
	}

	var items []map[string]attr.Value
	for _, cluster := range clusters {
		cpuObj, diags := cpuObjectValue(cluster.CPU)
		resp.Diagnostics.Append(diags...)
		gpuObj, diags := gpuObjectValue(cluster.GPU)
		resp.Diagnostics.Append(diags...)
		memoryObj, diags := memoryObjectValue(cluster.Memory)
		resp.Diagnostics.Append(diags...)
		gpuMemoryObj, diags := memoryObjectValue(cluster.GPUMemory)
		resp.Diagnostics.Append(diags...)
		storageObj, diags := storageObjectValue(cluster.Storage)
		resp.Diagnostics.Append(diags...)
		sshKeys, diags := stringListValue(ctx, cluster.SSHKeyIDs)
		resp.Diagnostics.Append(diags...)

		items = append(items, map[string]attr.Value{
			"id":                types.StringValue(cluster.ID),
			"cluster_type":      types.StringValue(cluster.ClusterType),
			"image":             types.StringValue(cluster.Image),
			"price_per_hour":    types.Float64Value(cluster.PricePerHour.Float64()),
			"hostname":          types.StringValue(cluster.Hostname),
			"description":       types.StringValue(cluster.Description),
			"ip":                stringPointerValueOrNull(cluster.IP),
			"status":            types.StringValue(cluster.Status),
			"created_at":        types.StringValue(cluster.CreatedAt.Format("2006-01-02T15:04:05Z")),
			"ssh_key_ids":       sshKeys,
			"cpu":               cpuObj,
			"gpu":               gpuObj,
			"memory":            memoryObj,
			"gpu_memory":        gpuMemoryObj,
			"storage":           storageObj,
			"location":          types.StringValue(cluster.Location),
			"os_name":           types.StringValue(cluster.OSName),
			"startup_script_id": stringPointerValueOrNull(cluster.StartupScriptID),
			"contract":          types.StringValue(cluster.Contract),
			"pricing":           types.StringValue(cluster.Pricing),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Clusters = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Container deployments data source

type ContainerDeploymentsDataSource struct {
	client *verda.Client
}

type ContainerDeploymentsDataSourceModel struct {
	Deployments types.List `tfsdk:"deployments"`
}

func NewContainerDeploymentsDataSource() datasource.DataSource {
	return &ContainerDeploymentsDataSource{}
}

func (d *ContainerDeploymentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_deployments"
}

func (d *ContainerDeploymentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches serverless container deployments.",
		Attributes: map[string]schema.Attribute{
			"deployments": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Container deployments.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"endpoint_base_url": schema.StringAttribute{
							Computed: true,
						},
						"created_at": schema.StringAttribute{Computed: true},
						"is_spot": schema.BoolAttribute{
							Computed: true,
						},
						"compute": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{Computed: true},
								"size": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
						"container_registry_settings": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"is_private": schema.BoolAttribute{Computed: true},
								"credentials": schema.StringAttribute{
									Computed: true,
								},
							},
						},
						"containers": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name":  schema.StringAttribute{Computed: true},
									"image": schema.StringAttribute{Computed: true},
									"exposed_port": schema.Int64Attribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *ContainerDeploymentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContainerDeploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerDeploymentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deployments, err := d.client.ContainerDeployments.GetDeployments(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container deployments, got error: %s", err))
		return
	}

	containerAttrTypes := map[string]attr.Type{
		"name":         types.StringType,
		"image":        types.StringType,
		"exposed_port": types.Int64Type,
	}

	deploymentAttrTypes := map[string]attr.Type{
		"name":              types.StringType,
		"endpoint_base_url": types.StringType,
		"created_at":        types.StringType,
		"is_spot":           types.BoolType,
		"compute": types.ObjectType{AttrTypes: map[string]attr.Type{
			"name": types.StringType,
			"size": types.Int64Type,
		}},
		"container_registry_settings": types.ObjectType{AttrTypes: map[string]attr.Type{
			"is_private":  types.BoolType,
			"credentials": types.StringType,
		}},
		"containers": types.ListType{
			ElemType: types.ObjectType{AttrTypes: containerAttrTypes},
		},
	}

	var deploymentItems []map[string]attr.Value
	for _, deployment := range deployments {
		var computeObj types.Object
		if deployment.Compute != nil {
			co, diags := types.ObjectValue(
				map[string]attr.Type{
					"name": types.StringType,
					"size": types.Int64Type,
				},
				map[string]attr.Value{
					"name": types.StringValue(deployment.Compute.Name),
					"size": types.Int64Value(int64(deployment.Compute.Size)),
				},
			)
			resp.Diagnostics.Append(diags...)
			computeObj = co
		} else {
			computeObj = types.ObjectNull(map[string]attr.Type{
				"name": types.StringType,
				"size": types.Int64Type,
			})
		}

		var registryObj types.Object
		if deployment.ContainerRegistrySettings != nil {
			credentials := types.StringNull()
			if deployment.ContainerRegistrySettings.Credentials != nil {
				credentials = types.StringValue(deployment.ContainerRegistrySettings.Credentials.Name)
			}
			ro, diags := types.ObjectValue(
				map[string]attr.Type{
					"is_private":  types.BoolType,
					"credentials": types.StringType,
				},
				map[string]attr.Value{
					"is_private":  types.BoolValue(deployment.ContainerRegistrySettings.IsPrivate),
					"credentials": credentials,
				},
			)
			resp.Diagnostics.Append(diags...)
			registryObj = ro
		} else {
			registryObj = types.ObjectNull(map[string]attr.Type{
				"is_private":  types.BoolType,
				"credentials": types.StringType,
			})
		}

		var containerItems []map[string]attr.Value
		for _, container := range deployment.Containers {
			containerItems = append(containerItems, map[string]attr.Value{
				"name":         types.StringValue(container.Name),
				"image":        types.StringValue(container.Image.Image),
				"exposed_port": types.Int64Value(int64(container.ExposedPort)),
			})
		}
		containersList, diags := objectListValue(containerAttrTypes, containerItems)
		resp.Diagnostics.Append(diags...)

		deploymentItems = append(deploymentItems, map[string]attr.Value{
			"name":                        types.StringValue(deployment.Name),
			"endpoint_base_url":           types.StringValue(deployment.EndpointBaseURL),
			"created_at":                  types.StringValue(deployment.CreatedAt.Format("2006-01-02T15:04:05Z")),
			"is_spot":                     types.BoolValue(deployment.IsSpot),
			"compute":                     computeObj,
			"container_registry_settings": registryObj,
			"containers":                  containersList,
		})
	}

	listValue, diags := objectListValue(deploymentAttrTypes, deploymentItems)
	resp.Diagnostics.Append(diags...)
	data.Deployments = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Job deployments data source

type JobDeploymentsDataSource struct {
	client *verda.Client
}

type JobDeploymentsDataSourceModel struct {
	Deployments types.List `tfsdk:"deployments"`
}

func NewJobDeploymentsDataSource() datasource.DataSource {
	return &JobDeploymentsDataSource{}
}

func (d *JobDeploymentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_deployments"
}

func (d *JobDeploymentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches serverless job deployments.",
		Attributes: map[string]schema.Attribute{
			"deployments": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Job deployments.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"compute": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{Computed: true},
								"size": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *JobDeploymentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *JobDeploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data JobDeploymentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deployments, err := d.client.ServerlessJobs.GetJobDeployments(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read job deployments, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"name":       types.StringType,
		"created_at": types.StringType,
		"compute": types.ObjectType{AttrTypes: map[string]attr.Type{
			"name": types.StringType,
			"size": types.Int64Type,
		}},
	}

	var items []map[string]attr.Value
	for _, deployment := range deployments {
		var computeObj types.Object
		if deployment.Compute != nil {
			co, diags := types.ObjectValue(
				map[string]attr.Type{
					"name": types.StringType,
					"size": types.Int64Type,
				},
				map[string]attr.Value{
					"name": types.StringValue(deployment.Compute.Name),
					"size": types.Int64Value(int64(deployment.Compute.Size)),
				},
			)
			resp.Diagnostics.Append(diags...)
			computeObj = co
		} else {
			computeObj = types.ObjectNull(map[string]attr.Type{
				"name": types.StringType,
				"size": types.Int64Type,
			})
		}

		items = append(items, map[string]attr.Value{
			"name":       types.StringValue(deployment.Name),
			"created_at": types.StringValue(deployment.CreatedAt),
			"compute":    computeObj,
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Deployments = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Container registry credentials data source

type ContainerRegistryCredentialsDataSource struct {
	client *verda.Client
}

type ContainerRegistryCredentialsDataSourceModel struct {
	Credentials types.List `tfsdk:"credentials"`
}

func NewContainerRegistryCredentialsDataSource() datasource.DataSource {
	return &ContainerRegistryCredentialsDataSource{}
}

func (d *ContainerRegistryCredentialsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_credentials"
}

func (d *ContainerRegistryCredentialsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches container registry credentials.",
		Attributes: map[string]schema.Attribute{
			"credentials": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Registry credentials.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *ContainerRegistryCredentialsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContainerRegistryCredentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerRegistryCredentialsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentials, err := d.client.ContainerDeployments.GetRegistryCredentials(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read registry credentials, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"name":       types.StringType,
		"created_at": types.StringType,
	}

	var items []map[string]attr.Value
	for _, credential := range credentials {
		items = append(items, map[string]attr.Value{
			"name":       types.StringValue(credential.Name),
			"created_at": types.StringValue(credential.CreatedAt),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Credentials = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// SSH keys data source

type SSHKeysDataSource struct {
	client *verda.Client
}

type SSHKeysDataSourceModel struct {
	Keys types.List `tfsdk:"keys"`
}

func NewSSHKeysDataSource() datasource.DataSource {
	return &SSHKeysDataSource{}
}

func (d *SSHKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (d *SSHKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches SSH keys.",
		Attributes: map[string]schema.Attribute{
			"keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "SSH keys.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"public_key": schema.StringAttribute{
							Computed: true,
						},
						"fingerprint": schema.StringAttribute{
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *SSHKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SSHKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SSHKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, err := d.client.SSHKeys.GetAllSSHKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SSH keys, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"public_key":  types.StringType,
		"fingerprint": types.StringType,
		"created_at":  types.StringType,
	}

	var items []map[string]attr.Value
	for _, key := range keys {
		items = append(items, map[string]attr.Value{
			"id":          types.StringValue(key.ID),
			"name":        types.StringValue(key.Name),
			"public_key":  types.StringValue(key.PublicKey),
			"fingerprint": types.StringValue(key.Fingerprint),
			"created_at":  types.StringValue(key.CreatedAt.Format("2006-01-02T15:04:05Z")),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Keys = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Startup scripts data source

type StartupScriptsDataSource struct {
	client *verda.Client
}

type StartupScriptsDataSourceModel struct {
	Scripts types.List `tfsdk:"scripts"`
}

func NewStartupScriptsDataSource() datasource.DataSource {
	return &StartupScriptsDataSource{}
}

func (d *StartupScriptsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_startup_scripts"
}

func (d *StartupScriptsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches startup scripts.",
		Attributes: map[string]schema.Attribute{
			"scripts": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Startup scripts.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"script": schema.StringAttribute{
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *StartupScriptsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *StartupScriptsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data StartupScriptsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scripts, err := d.client.StartupScripts.GetAllStartupScripts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read startup scripts, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":         types.StringType,
		"name":       types.StringType,
		"script":     types.StringType,
		"created_at": types.StringType,
	}

	var items []map[string]attr.Value
	for _, script := range scripts {
		items = append(items, map[string]attr.Value{
			"id":         types.StringValue(script.ID),
			"name":       types.StringValue(script.Name),
			"script":     types.StringValue(script.Script),
			"created_at": types.StringValue(script.CreatedAt.Format("2006-01-02T15:04:05Z")),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Scripts = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Secrets data source

type SecretsDataSource struct {
	client *verda.Client
}

type SecretsDataSourceModel struct {
	Secrets types.List `tfsdk:"secrets"`
}

type secretListResponse struct {
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	SecretType string `json:"secret_type"`
}

func NewSecretsDataSource() datasource.DataSource {
	return &SecretsDataSource{}
}

func (d *SecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secrets"
}

func (d *SecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches secrets.",
		Attributes: map[string]schema.Attribute{
			"secrets": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Secrets.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"secret_type": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *SecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SecretsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var response []secretListResponse
	if err := doVerdaRequest(ctx, d.client, "GET", "/secrets", nil, &response); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read secrets, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"name":        types.StringType,
		"created_at":  types.StringType,
		"secret_type": types.StringType,
	}

	var items []map[string]attr.Value
	for _, secret := range response {
		items = append(items, map[string]attr.Value{
			"name":        types.StringValue(secret.Name),
			"created_at":  types.StringValue(secret.CreatedAt),
			"secret_type": types.StringValue(secret.SecretType),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Secrets = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// File secrets data source

type FileSecretsDataSource struct {
	client *verda.Client
}

type FileSecretsDataSourceModel struct {
	Secrets types.List `tfsdk:"secrets"`
}

type fileSecretListResponse struct {
	Name       string   `json:"name"`
	CreatedAt  string   `json:"created_at"`
	SecretType string   `json:"secret_type"`
	FileNames  []string `json:"file_names"`
}

func NewFileSecretsDataSource() datasource.DataSource {
	return &FileSecretsDataSource{}
}

func (d *FileSecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_secrets"
}

func (d *FileSecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches file secrets.",
		Attributes: map[string]schema.Attribute{
			"secrets": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "File secrets.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"secret_type": schema.StringAttribute{
							Computed: true,
						},
						"file_names": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *FileSecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FileSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FileSecretsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var response []fileSecretListResponse
	if err := doVerdaRequest(ctx, d.client, "GET", "/file-secrets", nil, &response); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read file secrets, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"name":        types.StringType,
		"created_at":  types.StringType,
		"secret_type": types.StringType,
		"file_names":  types.ListType{ElemType: types.StringType},
	}

	var items []map[string]attr.Value
	for _, secret := range response {
		fileNames, diags := stringListValue(ctx, secret.FileNames)
		resp.Diagnostics.Append(diags...)
		items = append(items, map[string]attr.Value{
			"name":        types.StringValue(secret.Name),
			"created_at":  types.StringValue(secret.CreatedAt),
			"secret_type": types.StringValue(secret.SecretType),
			"file_names":  fileNames,
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Secrets = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
