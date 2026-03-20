package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var (
	_ datasource.DataSource = &ServerlessJobDataSource{}
	_ datasource.DataSource = &ServerlessJobsDataSource{}
	_ datasource.DataSource = &ServerlessJobScalingDataSource{}
	_ datasource.DataSource = &ServerlessJobStatusDataSource{}
)

func NewServerlessJobDataSource() datasource.DataSource {
	return &ServerlessJobDataSource{}
}

func NewServerlessJobsDataSource() datasource.DataSource {
	return &ServerlessJobsDataSource{}
}

func NewServerlessJobScalingDataSource() datasource.DataSource {
	return &ServerlessJobScalingDataSource{}
}

func NewServerlessJobStatusDataSource() datasource.DataSource {
	return &ServerlessJobStatusDataSource{}
}

type ServerlessJobDataSource struct {
	client *verda.Client
}

type ServerlessJobsDataSource struct {
	client *verda.Client
}

type ServerlessJobScalingDataSource struct {
	client *verda.Client
}

type ServerlessJobStatusDataSource struct {
	client *verda.Client
}

type ServerlessJobsDataSourceModel struct {
	Jobs types.List `tfsdk:"jobs"`
}

type ServerlessJobScalingDataSourceModel struct {
	Name                   types.String `tfsdk:"name"`
	MaxReplicaCount        types.Int64  `tfsdk:"max_replica_count"`
	QueueMessageTTLSeconds types.Int64  `tfsdk:"queue_message_ttl_seconds"`
	DeadlineSeconds        types.Int64  `tfsdk:"deadline_seconds"`
}

type ServerlessJobStatusDataSourceModel struct {
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

func configureServerlessJobDataSource(providerData any, diagnostics *diag.Diagnostics) *verda.Client {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*verda.Client)
	if !ok {
		diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *verda.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

func (d *ServerlessJobDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_job"
}

func (d *ServerlessJobsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_jobs"
}

func (d *ServerlessJobScalingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_job_scaling"
}

func (d *ServerlessJobStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_job_status"
}

func (d *ServerlessJobDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Reads a Verda serverless job deployment by name.",
		Attributes: map[string]datasourceschema.Attribute{
			"name": datasourceschema.StringAttribute{
				MarkdownDescription: "Name of the serverless job deployment.",
				Required:            true,
			},
			"compute": datasourceschema.SingleNestedAttribute{
				MarkdownDescription: "Compute resources for the job deployment.",
				Computed:            true,
				Attributes: map[string]datasourceschema.Attribute{
					"name": datasourceschema.StringAttribute{
						MarkdownDescription: "GPU type.",
						Computed:            true,
					},
					"size": datasourceschema.Int64Attribute{
						MarkdownDescription: "Number of GPUs.",
						Computed:            true,
					},
				},
			},
			"scaling": datasourceschema.SingleNestedAttribute{
				MarkdownDescription: "Scaling configuration for the job deployment.",
				Computed:            true,
				Attributes: map[string]datasourceschema.Attribute{
					"max_replica_count": datasourceschema.Int64Attribute{
						MarkdownDescription: "Maximum number of replicas.",
						Computed:            true,
					},
					"queue_message_ttl_seconds": datasourceschema.Int64Attribute{
						MarkdownDescription: "Queue message TTL in seconds.",
						Computed:            true,
					},
					"deadline_seconds": datasourceschema.Int64Attribute{
						MarkdownDescription: "Request deadline in seconds.",
						Computed:            true,
					},
				},
			},
			"container_registry_settings": datasourceschema.SingleNestedAttribute{
				MarkdownDescription: "Container registry authentication settings.",
				Computed:            true,
				Attributes: map[string]datasourceschema.Attribute{
					"is_private": datasourceschema.StringAttribute{
						MarkdownDescription: "Whether the registry is private ('true' or 'false').",
						Computed:            true,
					},
					"credentials": datasourceschema.StringAttribute{
						MarkdownDescription: "Name of the registry credentials resource.",
						Computed:            true,
					},
				},
			},
			"containers": datasourceschema.ListNestedAttribute{
				MarkdownDescription: "List of containers in the job deployment.",
				Computed:            true,
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"image": datasourceschema.StringAttribute{
							MarkdownDescription: "Container image.",
							Computed:            true,
						},
						"exposed_port": datasourceschema.Int64Attribute{
							MarkdownDescription: "Port exposed by the container.",
							Computed:            true,
						},
						"healthcheck": datasourceschema.SingleNestedAttribute{
							MarkdownDescription: "Healthcheck configuration.",
							Computed:            true,
							Attributes: map[string]datasourceschema.Attribute{
								"enabled": datasourceschema.StringAttribute{
									MarkdownDescription: "Whether healthcheck is enabled ('true' or 'false').",
									Computed:            true,
								},
								"port": datasourceschema.StringAttribute{
									MarkdownDescription: "Port for healthcheck.",
									Computed:            true,
								},
								"path": datasourceschema.StringAttribute{
									MarkdownDescription: "Path for healthcheck.",
									Computed:            true,
								},
							},
						},
						"entrypoint_overrides": datasourceschema.SingleNestedAttribute{
							MarkdownDescription: "Override container entrypoint and command.",
							Computed:            true,
							Attributes: map[string]datasourceschema.Attribute{
								"enabled": datasourceschema.BoolAttribute{
									MarkdownDescription: "Whether to override the entrypoint.",
									Computed:            true,
								},
								"entrypoint": datasourceschema.ListAttribute{
									MarkdownDescription: "Custom entrypoint array.",
									ElementType:         types.StringType,
									Computed:            true,
								},
								"cmd": datasourceschema.ListAttribute{
									MarkdownDescription: "Custom command array.",
									ElementType:         types.StringType,
									Computed:            true,
								},
							},
						},
						"env": datasourceschema.ListNestedAttribute{
							MarkdownDescription: "Environment variables.",
							Computed:            true,
							NestedObject: datasourceschema.NestedAttributeObject{
								Attributes: map[string]datasourceschema.Attribute{
									"type": datasourceschema.StringAttribute{
										MarkdownDescription: "Type of environment variable ('plain' or 'secret').",
										Computed:            true,
									},
									"name": datasourceschema.StringAttribute{
										MarkdownDescription: "Name of the environment variable.",
										Computed:            true,
									},
									"value_or_reference_to_secret": datasourceschema.StringAttribute{
										MarkdownDescription: "Value for plain env vars or secret name for secret env vars.",
										Computed:            true,
									},
								},
							},
						},
						"volume_mounts": datasourceschema.ListNestedAttribute{
							MarkdownDescription: "Volume mounts for the container.",
							Computed:            true,
							NestedObject: datasourceschema.NestedAttributeObject{
								Attributes: map[string]datasourceschema.Attribute{
									"type": datasourceschema.StringAttribute{
										MarkdownDescription: "Type of volume ('scratch', 'memory', 'secret', 'shared').",
										Computed:            true,
									},
									"mount_path": datasourceschema.StringAttribute{
										MarkdownDescription: "Path where volume will be mounted in container.",
										Computed:            true,
									},
									"secret_name": datasourceschema.StringAttribute{
										MarkdownDescription: "Name of secret when present.",
										Computed:            true,
									},
									"size_in_mb": datasourceschema.Int64Attribute{
										MarkdownDescription: "Size in MB when present.",
										Computed:            true,
									},
									"volume_id": datasourceschema.StringAttribute{
										MarkdownDescription: "Volume ID when present.",
										Computed:            true,
									},
								},
							},
						},
					},
				},
			},
			"endpoint_base_url": datasourceschema.StringAttribute{
				MarkdownDescription: "Base URL for the job deployment endpoint.",
				Computed:            true,
			},
			"created_at": datasourceschema.StringAttribute{
				MarkdownDescription: "Timestamp when the job deployment was created.",
				Computed:            true,
			},
		},
	}
}

func (d *ServerlessJobsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Lists Verda serverless job deployments.",
		Attributes: map[string]datasourceschema.Attribute{
			"jobs": datasourceschema.ListNestedAttribute{
				MarkdownDescription: "Serverless job deployments returned by the API.",
				Computed:            true,
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"name": datasourceschema.StringAttribute{
							MarkdownDescription: "Name of the serverless job deployment.",
							Computed:            true,
						},
						"created_at": datasourceschema.StringAttribute{
							MarkdownDescription: "Timestamp when the job deployment was created.",
							Computed:            true,
						},
						"compute": datasourceschema.SingleNestedAttribute{
							MarkdownDescription: "Compute resources for the job deployment.",
							Computed:            true,
							Attributes: map[string]datasourceschema.Attribute{
								"name": datasourceschema.StringAttribute{
									MarkdownDescription: "GPU type.",
									Computed:            true,
								},
								"size": datasourceschema.Int64Attribute{
									MarkdownDescription: "Number of GPUs.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *ServerlessJobScalingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Reads the scaling configuration for a Verda serverless job deployment.",
		Attributes: map[string]datasourceschema.Attribute{
			"name": datasourceschema.StringAttribute{
				MarkdownDescription: "Name of the serverless job deployment.",
				Required:            true,
			},
			"max_replica_count": datasourceschema.Int64Attribute{
				MarkdownDescription: "Maximum number of replicas.",
				Computed:            true,
			},
			"queue_message_ttl_seconds": datasourceschema.Int64Attribute{
				MarkdownDescription: "Queue message TTL in seconds.",
				Computed:            true,
			},
			"deadline_seconds": datasourceschema.Int64Attribute{
				MarkdownDescription: "Request deadline in seconds.",
				Computed:            true,
			},
		},
	}
}

func (d *ServerlessJobStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Reads the runtime status for a Verda serverless job deployment.",
		Attributes: map[string]datasourceschema.Attribute{
			"name": datasourceschema.StringAttribute{
				MarkdownDescription: "Name of the serverless job deployment.",
				Required:            true,
			},
			"status": datasourceschema.StringAttribute{
				MarkdownDescription: "Runtime status of the serverless job deployment.",
				Computed:            true,
			},
		},
	}
}

func (d *ServerlessJobDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureServerlessJobDataSource(req.ProviderData, &resp.Diagnostics)
}

func (d *ServerlessJobsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureServerlessJobDataSource(req.ProviderData, &resp.Diagnostics)
}

func (d *ServerlessJobScalingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureServerlessJobDataSource(req.ProviderData, &resp.Diagnostics)
}

func (d *ServerlessJobStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureServerlessJobDataSource(req.ProviderData, &resp.Diagnostics)
}

func (d *ServerlessJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerlessJobResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deployment, err := d.client.ServerlessJobs.GetJobDeploymentByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read serverless job deployment, got error: %s", err))
		return
	}

	flattenJobDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ServerlessJobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerlessJobsDataSourceModel

	jobs, err := d.client.ServerlessJobs.GetJobDeployments(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list serverless job deployments, got error: %s", err))
		return
	}

	data.Jobs = flattenJobDeploymentShortInfos(ctx, jobs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ServerlessJobScalingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerlessJobScalingDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scaling, err := d.client.ServerlessJobs.GetJobDeploymentScaling(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read serverless job deployment scaling, got error: %s", err))
		return
	}

	data.MaxReplicaCount = types.Int64Value(int64(scaling.MaxReplicaCount))
	data.QueueMessageTTLSeconds = types.Int64Value(int64(scaling.QueueMessageTTLSeconds))
	data.DeadlineSeconds = types.Int64Value(int64(scaling.DeadlineSeconds))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ServerlessJobStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerlessJobStatusDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, err := d.client.ServerlessJobs.GetJobDeploymentStatus(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read serverless job deployment status, got error: %s", err))
		return
	}

	data.Status = types.StringValue(status.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
