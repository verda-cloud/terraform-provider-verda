package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &ServerlessJobResource{}
var _ resource.ResourceWithImportState = &ServerlessJobResource{}

func NewServerlessJobResource() resource.Resource {
	return &ServerlessJobResource{}
}

type ServerlessJobResource struct {
	client *verda.Client
}

type ServerlessJobResourceModel struct {
	Name                      types.String `tfsdk:"name"`
	Compute                   types.Object `tfsdk:"compute"`
	Scaling                   types.Object `tfsdk:"scaling"`
	ContainerRegistrySettings types.Object `tfsdk:"container_registry_settings"`
	Containers                types.List   `tfsdk:"containers"`
	EndpointBaseURL           types.String `tfsdk:"endpoint_base_url"`
	CreatedAt                 types.String `tfsdk:"created_at"`
}

type JobScalingModel struct {
	MaxReplicaCount        types.Int64 `tfsdk:"max_replica_count"`
	QueueMessageTTLSeconds types.Int64 `tfsdk:"queue_message_ttl_seconds"`
	DeadlineSeconds        types.Int64 `tfsdk:"deadline_seconds"`
}

func (r *ServerlessJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_job"
}

func (r *ServerlessJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda serverless job deployment",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the serverless job deployment",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"compute": schema.SingleNestedAttribute{
				MarkdownDescription: "Compute resources for the job deployment",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "GPU type (e.g., 'H100', 'A100')",
						Required:            true,
					},
					"size": schema.Int64Attribute{
						MarkdownDescription: "Number of GPUs",
						Required:            true,
					},
				},
			},
			"scaling": schema.SingleNestedAttribute{
				MarkdownDescription: "Scaling configuration for the job deployment",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"max_replica_count": schema.Int64Attribute{
						MarkdownDescription: "Maximum number of replicas",
						Required:            true,
					},
					"queue_message_ttl_seconds": schema.Int64Attribute{
						MarkdownDescription: "Queue message TTL in seconds",
						Required:            true,
					},
					"deadline_seconds": schema.Int64Attribute{
						MarkdownDescription: "Request deadline in seconds",
						Optional:            true,
					},
				},
			},
			"container_registry_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Container registry authentication settings",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"is_private": schema.StringAttribute{
						MarkdownDescription: "Whether the registry is private ('true' or 'false')",
						Optional:            true,
						Computed:            true,
					},
					"credentials": schema.StringAttribute{
						MarkdownDescription: "Name of the registry credentials resource",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"containers": schema.ListNestedAttribute{
				MarkdownDescription: "List of containers in the job deployment",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"image": schema.StringAttribute{
							MarkdownDescription: "Container image (e.g., 'nginx:latest')",
							Required:            true,
						},
						"exposed_port": schema.Int64Attribute{
							MarkdownDescription: "Port exposed by the container",
							Required:            true,
						},
						"healthcheck": schema.SingleNestedAttribute{
							MarkdownDescription: "Healthcheck configuration",
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.StringAttribute{
									MarkdownDescription: "Whether healthcheck is enabled ('true' or 'false')",
									Required:            true,
								},
								"port": schema.StringAttribute{
									MarkdownDescription: "Port for healthcheck",
									Optional:            true,
								},
								"path": schema.StringAttribute{
									MarkdownDescription: "Path for healthcheck",
									Optional:            true,
								},
							},
						},
						"entrypoint_overrides": schema.SingleNestedAttribute{
							MarkdownDescription: "Override container entrypoint and command",
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									MarkdownDescription: "Whether to override the entrypoint",
									Required:            true,
								},
								"entrypoint": schema.ListAttribute{
									MarkdownDescription: "Custom entrypoint array (e.g., [\"/bin/sh\", \"-c\"])",
									ElementType:         types.StringType,
									Optional:            true,
								},
								"cmd": schema.ListAttribute{
									MarkdownDescription: "Custom command array",
									ElementType:         types.StringType,
									Optional:            true,
								},
							},
						},
						"env": schema.ListNestedAttribute{
							MarkdownDescription: "Environment variables",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										MarkdownDescription: "Type of environment variable ('plain' or 'secret')",
										Required:            true,
									},
									"name": schema.StringAttribute{
										MarkdownDescription: "Name of the environment variable",
										Required:            true,
									},
									"value_or_reference_to_secret": schema.StringAttribute{
										MarkdownDescription: "Value for plain env vars or secret name for secret env vars",
										Required:            true,
									},
								},
							},
						},
						"volume_mounts": schema.ListNestedAttribute{
							MarkdownDescription: "Volume mounts for the container",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										MarkdownDescription: "Type of volume ('scratch', 'memory', 'secret', 'shared')",
										Required:            true,
									},
									"mount_path": schema.StringAttribute{
										MarkdownDescription: "Path where volume will be mounted in container",
										Required:            true,
									},
									"secret_name": schema.StringAttribute{
										MarkdownDescription: "Name of secret (required for type='secret')",
										Optional:            true,
									},
									"size_in_mb": schema.Int64Attribute{
										MarkdownDescription: "Size in MB (optional for type='scratch' or 'memory')",
										Optional:            true,
									},
									"volume_id": schema.StringAttribute{
										MarkdownDescription: "Volume ID (required for type='shared')",
										Optional:            true,
									},
								},
								Validators: []validator.Object{
									volumeMountValidator{},
								},
							},
						},
					},
				},
			},
			"endpoint_base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL for the job deployment endpoint",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the job deployment was created",
				Computed:            true,
			},
		},
	}
}

func (r *ServerlessJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerlessJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServerlessJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := buildServerlessJobCreateRequest(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || createReq == nil {
		return
	}

	deployment, err := r.client.ServerlessJobs.CreateJobDeployment(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create serverless job deployment, got error: %s", err))
		return
	}

	// Flatten API response, merging with plan to preserve fields the API doesn't return
	planContainers := data.Containers
	flattenJobDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	mergeJobContainersFromPlan(ctx, planContainers, &data, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerlessJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServerlessJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deployment, err := r.client.ServerlessJobs.GetJobDeploymentByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read serverless job deployment, got error: %s", err))
		return
	}

	// Preserve container configuration from prior state
	priorContainers := data.Containers
	flattenJobDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	mergeJobContainersFromPlan(ctx, priorContainers, &data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerlessJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServerlessJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := buildServerlessJobUpdateRequest(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || updateReq == nil {
		return
	}

	deployment, err := r.client.ServerlessJobs.UpdateJobDeployment(ctx, data.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update serverless job deployment, got error: %s", err))
		return
	}

	planContainers := data.Containers
	flattenJobDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	mergeJobContainersFromPlan(ctx, planContainers, &data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerlessJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServerlessJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ServerlessJobs.DeleteJobDeployment(ctx, data.Name.ValueString(), 300000)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete serverless job deployment, got error: %s", err))
		return
	}
}

func (r *ServerlessJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
