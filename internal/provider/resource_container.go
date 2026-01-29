package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &ContainerResource{}
var _ resource.ResourceWithImportState = &ContainerResource{}

func NewContainerResource() resource.Resource {
	return &ContainerResource{}
}

type ContainerResource struct {
	client *verda.Client
}

type ContainerResourceModel struct {
	Name                      types.String `tfsdk:"name"`
	IsSpot                    types.Bool   `tfsdk:"is_spot"`
	Compute                   types.Object `tfsdk:"compute"`
	Scaling                   types.Object `tfsdk:"scaling"`
	ContainerRegistrySettings types.Object `tfsdk:"container_registry_settings"`
	Containers                types.List   `tfsdk:"containers"`
	EndpointBaseURL           types.String `tfsdk:"endpoint_base_url"`
	CreatedAt                 types.String `tfsdk:"created_at"`
}

type ComputeModel struct {
	Name types.String `tfsdk:"name"`
	Size types.Int64  `tfsdk:"size"`
}

type ScalingModel struct {
	MinReplicaCount              types.Int64  `tfsdk:"min_replica_count"`
	MaxReplicaCount              types.Int64  `tfsdk:"max_replica_count"`
	QueueMessageTTLSeconds       types.Int64  `tfsdk:"queue_message_ttl_seconds"`
	DeadlineSeconds              types.Int64  `tfsdk:"deadline_seconds"`
	ConcurrentRequestsPerReplica types.Int64  `tfsdk:"concurrent_requests_per_replica"`
	ScaleDownPolicy              types.Object `tfsdk:"scale_down_policy"`
	ScaleUpPolicy                types.Object `tfsdk:"scale_up_policy"`
	QueueLoad                    types.Object `tfsdk:"queue_load"`
}

type ScalingPolicyModel struct {
	DelaySeconds types.Int64 `tfsdk:"delay_seconds"`
}

type QueueLoadTriggerModel struct {
	Threshold types.Float64 `tfsdk:"threshold"`
}

type RegistrySettingsModel struct {
	IsPrivate   types.String `tfsdk:"is_private"`
	Credentials types.String `tfsdk:"credentials"`
}

type ContainerModel struct {
	Image               types.String `tfsdk:"image"`
	ExposedPort         types.Int64  `tfsdk:"exposed_port"`
	Healthcheck         types.Object `tfsdk:"healthcheck"`
	EntrypointOverrides types.Object `tfsdk:"entrypoint_overrides"`
	Env                 types.List   `tfsdk:"env"`
	VolumeMounts        types.List   `tfsdk:"volume_mounts"`
}

type HealthcheckModel struct {
	Enabled types.String `tfsdk:"enabled"`
	Port    types.String `tfsdk:"port"`
	Path    types.String `tfsdk:"path"`
}

type EnvVarModel struct {
	Type                     types.String `tfsdk:"type"`
	Name                     types.String `tfsdk:"name"`
	ValueOrReferenceToSecret types.String `tfsdk:"value_or_reference_to_secret"`
}

type EntrypointOverridesModel struct {
	Enabled    types.Bool `tfsdk:"enabled"`
	Entrypoint types.List `tfsdk:"entrypoint"`
	Cmd        types.List `tfsdk:"cmd"`
}

type VolumeMountModel struct {
	Type       types.String `tfsdk:"type"`
	MountPath  types.String `tfsdk:"mount_path"`
	SecretName types.String `tfsdk:"secret_name"`
	SizeInMB   types.Int64  `tfsdk:"size_in_mb"`
	VolumeID   types.String `tfsdk:"volume_id"`
}

func (r *ContainerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *ContainerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda container deployment for serverless workloads",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the container deployment",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_spot": schema.BoolAttribute{
				MarkdownDescription: "Whether to use spot instances (defaults to false)",
				Optional:            true,
				Computed:            true,
			},
			"compute": schema.SingleNestedAttribute{
				MarkdownDescription: "Compute resources for the deployment",
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
				MarkdownDescription: "Scaling configuration for the deployment",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"min_replica_count": schema.Int64Attribute{
						MarkdownDescription: "Minimum number of replicas",
						Required:            true,
					},
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
					"concurrent_requests_per_replica": schema.Int64Attribute{
						MarkdownDescription: "Maximum concurrent requests per replica",
						Required:            true,
					},
					"scale_down_policy": schema.SingleNestedAttribute{
						MarkdownDescription: "Scale down policy configuration",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"delay_seconds": schema.Int64Attribute{
								MarkdownDescription: "Delay in seconds before scaling down",
								Required:            true,
							},
						},
					},
					"scale_up_policy": schema.SingleNestedAttribute{
						MarkdownDescription: "Scale up policy configuration",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"delay_seconds": schema.Int64Attribute{
								MarkdownDescription: "Delay in seconds before scaling up",
								Required:            true,
							},
						},
					},
					"queue_load": schema.SingleNestedAttribute{
						MarkdownDescription: "Queue load trigger configuration",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"threshold": schema.Float64Attribute{
								MarkdownDescription: "Queue load threshold for scaling",
								Required:            true,
							},
						},
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
				MarkdownDescription: "List of containers in the deployment",
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
				MarkdownDescription: "Base URL for the deployment endpoint",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the deployment was created",
				Computed:            true,
			},
		},
	}
}

func (r *ContainerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContainerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &verda.CreateDeploymentRequest{
		Name:   data.Name.ValueString(),
		IsSpot: data.IsSpot.ValueBool(),
	}

	// Parse compute
	var compute ComputeModel
	resp.Diagnostics.Append(data.Compute.As(ctx, &compute, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq.Compute = verda.ContainerCompute{
		Name: compute.Name.ValueString(),
		Size: int(compute.Size.ValueInt64()),
	}

	// Parse scaling
	var scaling ScalingModel
	resp.Diagnostics.Append(data.Scaling.As(ctx, &scaling, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	scalingOptions := verda.ContainerScalingOptions{
		MinReplicaCount:              int(scaling.MinReplicaCount.ValueInt64()),
		MaxReplicaCount:              int(scaling.MaxReplicaCount.ValueInt64()),
		QueueMessageTTLSeconds:       int(scaling.QueueMessageTTLSeconds.ValueInt64()),
		ConcurrentRequestsPerReplica: int(scaling.ConcurrentRequestsPerReplica.ValueInt64()),
	}

	// Parse scale down policy
	var scaleDownPolicy ScalingPolicyModel
	resp.Diagnostics.Append(scaling.ScaleDownPolicy.As(ctx, &scaleDownPolicy, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	scalingOptions.ScaleDownPolicy = &verda.ScalingPolicy{
		DelaySeconds: int(scaleDownPolicy.DelaySeconds.ValueInt64()),
	}

	// Parse scale up policy
	var scaleUpPolicy ScalingPolicyModel
	resp.Diagnostics.Append(scaling.ScaleUpPolicy.As(ctx, &scaleUpPolicy, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	scalingOptions.ScaleUpPolicy = &verda.ScalingPolicy{
		DelaySeconds: int(scaleUpPolicy.DelaySeconds.ValueInt64()),
	}

	// Parse queue load trigger
	var queueLoad QueueLoadTriggerModel
	resp.Diagnostics.Append(scaling.QueueLoad.As(ctx, &queueLoad, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	scalingOptions.ScalingTriggers = &verda.ScalingTriggers{
		QueueLoad: &verda.QueueLoadTrigger{
			Threshold: queueLoad.Threshold.ValueFloat64(),
		},
	}

	createReq.Scaling = scalingOptions

	// Parse container registry settings if provided
	if !data.ContainerRegistrySettings.IsNull() && !data.ContainerRegistrySettings.IsUnknown() {
		var registrySettings RegistrySettingsModel
		resp.Diagnostics.Append(data.ContainerRegistrySettings.As(ctx, &registrySettings, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		isPrivate := registrySettings.IsPrivate.ValueString() == "true"
		createReq.ContainerRegistrySettings = verda.ContainerRegistrySettings{
			IsPrivate: isPrivate,
		}

		if !registrySettings.Credentials.IsNull() && registrySettings.Credentials.ValueString() != "" {
			createReq.ContainerRegistrySettings.Credentials = &verda.RegistryCredentialsRef{
				Name: registrySettings.Credentials.ValueString(),
			}
		}
	} else {
		// By default, we'll assume the registry is public
		createReq.ContainerRegistrySettings = verda.ContainerRegistrySettings{
			IsPrivate: false,
		}
	}

	// Parse containers
	var containers []ContainerModel
	resp.Diagnostics.Append(data.Containers.ElementsAs(ctx, &containers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var deploymentContainers []verda.CreateDeploymentContainer
	for _, container := range containers {
		deploymentContainer := verda.CreateDeploymentContainer{
			Image:       container.Image.ValueString(),
			ExposedPort: int(container.ExposedPort.ValueInt64()),
		}

		// Parse healthcheck if provided
		if !container.Healthcheck.IsNull() {
			var healthcheck HealthcheckModel
			resp.Diagnostics.Append(container.Healthcheck.As(ctx, &healthcheck, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				return
			}

			enabled := healthcheck.Enabled.ValueString() == "true"
			hc := &verda.ContainerHealthcheck{
				Enabled: enabled,
			}

			if !healthcheck.Port.IsNull() && healthcheck.Port.ValueString() != "" {
				var port int
				_, err := fmt.Sscanf(healthcheck.Port.ValueString(), "%d", &port)
				if err == nil {
					hc.Port = port
				}
			}

			if !healthcheck.Path.IsNull() {
				hc.Path = healthcheck.Path.ValueString()
			}

			deploymentContainer.Healthcheck = hc
		}

		// Parse entrypoint overrides if provided
		if !container.EntrypointOverrides.IsNull() && !container.EntrypointOverrides.IsUnknown() {
			var entrypointOverrides EntrypointOverridesModel
			resp.Diagnostics.Append(container.EntrypointOverrides.As(ctx, &entrypointOverrides, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				return
			}

			overrides := &verda.ContainerEntrypointOverrides{
				Enabled: entrypointOverrides.Enabled.ValueBool(),
			}

			if !entrypointOverrides.Entrypoint.IsNull() && !entrypointOverrides.Entrypoint.IsUnknown() {
				var entrypoint []string
				resp.Diagnostics.Append(entrypointOverrides.Entrypoint.ElementsAs(ctx, &entrypoint, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
				overrides.Entrypoint = entrypoint
			}

			if !entrypointOverrides.Cmd.IsNull() && !entrypointOverrides.Cmd.IsUnknown() {
				var cmd []string
				resp.Diagnostics.Append(entrypointOverrides.Cmd.ElementsAs(ctx, &cmd, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
				overrides.Cmd = cmd
			}

			deploymentContainer.EntrypointOverrides = overrides
		}

		// Parse environment variables if provided
		if !container.Env.IsNull() {
			var envVars []EnvVarModel
			resp.Diagnostics.Append(container.Env.ElementsAs(ctx, &envVars, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			var containerEnvVars []verda.ContainerEnvVar
			for _, envVar := range envVars {
				containerEnvVars = append(containerEnvVars, verda.ContainerEnvVar{
					Type:                     envVar.Type.ValueString(),
					Name:                     envVar.Name.ValueString(),
					ValueOrReferenceToSecret: envVar.ValueOrReferenceToSecret.ValueString(),
				})
			}
			deploymentContainer.Env = containerEnvVars
		}

		// Parse volume mounts if provided
		if !container.VolumeMounts.IsNull() {
			var volumeMounts []VolumeMountModel
			resp.Diagnostics.Append(container.VolumeMounts.ElementsAs(ctx, &volumeMounts, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			var containerVolumeMounts []verda.ContainerVolumeMount
			for _, volumeMount := range volumeMounts {
				mount := verda.ContainerVolumeMount{
					Type:      volumeMount.Type.ValueString(),
					MountPath: volumeMount.MountPath.ValueString(),
				}

				if !volumeMount.SecretName.IsNull() && volumeMount.SecretName.ValueString() != "" {
					mount.SecretName = volumeMount.SecretName.ValueString()
				}

				if !volumeMount.SizeInMB.IsNull() {
					mount.SizeInMB = int(volumeMount.SizeInMB.ValueInt64())
				}

				if !volumeMount.VolumeID.IsNull() && volumeMount.VolumeID.ValueString() != "" {
					mount.VolumeID = volumeMount.VolumeID.ValueString()
				}

				containerVolumeMounts = append(containerVolumeMounts, mount)
			}
			deploymentContainer.VolumeMounts = containerVolumeMounts
		}

		deploymentContainers = append(deploymentContainers, deploymentContainer)
	}

	createReq.Containers = deploymentContainers

	deployment, err := r.client.ContainerDeployments.CreateDeployment(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create container deployment, got error: %s", err))
		return
	}

	// Flatten API response, merging with plan to preserve fields the API doesn't return
	planContainers := data.Containers
	r.flattenDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	// Merge API response with plan to preserve fields the API doesn't echo back
	r.mergeContainersFromPlan(ctx, planContainers, &data, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContainerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deployment, err := r.client.ContainerDeployments.GetDeploymentByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container deployment, got error: %s", err))
		return
	}

	// Also fetch scaling configuration
	scalingConfig, err := r.client.ContainerDeployments.GetDeploymentScaling(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scaling configuration, got error: %s", err))
		return
	}

	// Preserve container configuration from prior state
	// The API doesn't return all fields (like volume_id for non-shared volumes)
	priorContainers := data.Containers
	r.flattenDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	r.flattenScalingToModel(ctx, scalingConfig, &data, &resp.Diagnostics)
	// Merge API response with prior state to preserve fields the API doesn't return
	r.mergeContainersFromPlan(ctx, priorContainers, &data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ContainerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &verda.UpdateDeploymentRequest{}

	if !data.IsSpot.IsNull() && !data.IsSpot.IsUnknown() {
		isSpot := data.IsSpot.ValueBool()
		updateReq.IsSpot = &isSpot
	}

	// Parse compute
	var compute ComputeModel
	resp.Diagnostics.Append(data.Compute.As(ctx, &compute, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq.Compute = &verda.ContainerCompute{
		Name: compute.Name.ValueString(),
		Size: int(compute.Size.ValueInt64()),
	}

	// Parse scaling
	var scaling ScalingModel
	resp.Diagnostics.Append(data.Scaling.As(ctx, &scaling, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	scalingOptions := verda.ContainerScalingOptions{
		MinReplicaCount:              int(scaling.MinReplicaCount.ValueInt64()),
		MaxReplicaCount:              int(scaling.MaxReplicaCount.ValueInt64()),
		QueueMessageTTLSeconds:       int(scaling.QueueMessageTTLSeconds.ValueInt64()),
		ConcurrentRequestsPerReplica: int(scaling.ConcurrentRequestsPerReplica.ValueInt64()),
	}

	// Parse scale down policy
	var scaleDownPolicy ScalingPolicyModel
	resp.Diagnostics.Append(scaling.ScaleDownPolicy.As(ctx, &scaleDownPolicy, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	scalingOptions.ScaleDownPolicy = &verda.ScalingPolicy{
		DelaySeconds: int(scaleDownPolicy.DelaySeconds.ValueInt64()),
	}

	// Parse scale up policy
	var scaleUpPolicy ScalingPolicyModel
	resp.Diagnostics.Append(scaling.ScaleUpPolicy.As(ctx, &scaleUpPolicy, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	scalingOptions.ScaleUpPolicy = &verda.ScalingPolicy{
		DelaySeconds: int(scaleUpPolicy.DelaySeconds.ValueInt64()),
	}

	// Parse queue load trigger
	var queueLoad QueueLoadTriggerModel
	resp.Diagnostics.Append(scaling.QueueLoad.As(ctx, &queueLoad, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	scalingOptions.ScalingTriggers = &verda.ScalingTriggers{
		QueueLoad: &verda.QueueLoadTrigger{
			Threshold: queueLoad.Threshold.ValueFloat64(),
		},
	}

	// Preserve CPU/GPU utilization triggers if present
	currentScaling, err := r.client.ContainerDeployments.GetDeploymentScaling(ctx, data.Name.ValueString())
	if err == nil && currentScaling != nil && currentScaling.ScalingTriggers != nil {
		if currentScaling.ScalingTriggers.CPUUtilization != nil {
			scalingOptions.ScalingTriggers.CPUUtilization = currentScaling.ScalingTriggers.CPUUtilization
		}
		if currentScaling.ScalingTriggers.GPUUtilization != nil {
			scalingOptions.ScalingTriggers.GPUUtilization = currentScaling.ScalingTriggers.GPUUtilization
		}
	}

	updateReq.Scaling = &scalingOptions

	// Parse container registry settings if provided
	if !data.ContainerRegistrySettings.IsNull() && !data.ContainerRegistrySettings.IsUnknown() {
		var registrySettings RegistrySettingsModel
		resp.Diagnostics.Append(data.ContainerRegistrySettings.As(ctx, &registrySettings, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		isPrivate := registrySettings.IsPrivate.ValueString() == "true"
		updateReq.ContainerRegistrySettings = &verda.ContainerRegistrySettings{
			IsPrivate: isPrivate,
		}

		if !registrySettings.Credentials.IsNull() && registrySettings.Credentials.ValueString() != "" {
			updateReq.ContainerRegistrySettings.Credentials = &verda.RegistryCredentialsRef{
				Name: registrySettings.Credentials.ValueString(),
			}
		}
	}

	// Parse containers
	var containers []ContainerModel
	resp.Diagnostics.Append(data.Containers.ElementsAs(ctx, &containers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var deploymentContainers []verda.CreateDeploymentContainer
	for _, container := range containers {
		deploymentContainer := verda.CreateDeploymentContainer{
			Image:       container.Image.ValueString(),
			ExposedPort: int(container.ExposedPort.ValueInt64()),
		}

		// Parse healthcheck if provided
		if !container.Healthcheck.IsNull() {
			var healthcheck HealthcheckModel
			resp.Diagnostics.Append(container.Healthcheck.As(ctx, &healthcheck, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				return
			}

			enabled := healthcheck.Enabled.ValueString() == "true"
			hc := &verda.ContainerHealthcheck{
				Enabled: enabled,
			}

			if !healthcheck.Port.IsNull() && healthcheck.Port.ValueString() != "" {
				var port int
				_, err := fmt.Sscanf(healthcheck.Port.ValueString(), "%d", &port)
				if err == nil {
					hc.Port = port
				}
			}

			if !healthcheck.Path.IsNull() {
				hc.Path = healthcheck.Path.ValueString()
			}

			deploymentContainer.Healthcheck = hc
		}

		// Parse entrypoint overrides if provided
		if !container.EntrypointOverrides.IsNull() && !container.EntrypointOverrides.IsUnknown() {
			var entrypointOverrides EntrypointOverridesModel
			resp.Diagnostics.Append(container.EntrypointOverrides.As(ctx, &entrypointOverrides, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				return
			}

			overrides := &verda.ContainerEntrypointOverrides{
				Enabled: entrypointOverrides.Enabled.ValueBool(),
			}

			if !entrypointOverrides.Entrypoint.IsNull() && !entrypointOverrides.Entrypoint.IsUnknown() {
				var entrypoint []string
				resp.Diagnostics.Append(entrypointOverrides.Entrypoint.ElementsAs(ctx, &entrypoint, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
				overrides.Entrypoint = entrypoint
			}

			if !entrypointOverrides.Cmd.IsNull() && !entrypointOverrides.Cmd.IsUnknown() {
				var cmd []string
				resp.Diagnostics.Append(entrypointOverrides.Cmd.ElementsAs(ctx, &cmd, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
				overrides.Cmd = cmd
			}

			deploymentContainer.EntrypointOverrides = overrides
		}

		// Parse environment variables if provided
		if !container.Env.IsNull() {
			var envVars []EnvVarModel
			resp.Diagnostics.Append(container.Env.ElementsAs(ctx, &envVars, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			var containerEnvVars []verda.ContainerEnvVar
			for _, envVar := range envVars {
				containerEnvVars = append(containerEnvVars, verda.ContainerEnvVar{
					Type:                     envVar.Type.ValueString(),
					Name:                     envVar.Name.ValueString(),
					ValueOrReferenceToSecret: envVar.ValueOrReferenceToSecret.ValueString(),
				})
			}
			deploymentContainer.Env = containerEnvVars
		}

		// Parse volume mounts if provided
		if !container.VolumeMounts.IsNull() {
			var volumeMounts []VolumeMountModel
			resp.Diagnostics.Append(container.VolumeMounts.ElementsAs(ctx, &volumeMounts, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			var containerVolumeMounts []verda.ContainerVolumeMount
			for _, volumeMount := range volumeMounts {
				mount := verda.ContainerVolumeMount{
					Type:      volumeMount.Type.ValueString(),
					MountPath: volumeMount.MountPath.ValueString(),
				}

				if !volumeMount.SecretName.IsNull() && volumeMount.SecretName.ValueString() != "" {
					mount.SecretName = volumeMount.SecretName.ValueString()
				}

				if !volumeMount.SizeInMB.IsNull() {
					mount.SizeInMB = int(volumeMount.SizeInMB.ValueInt64())
				}

				if !volumeMount.VolumeID.IsNull() && volumeMount.VolumeID.ValueString() != "" {
					mount.VolumeID = volumeMount.VolumeID.ValueString()
				}

				containerVolumeMounts = append(containerVolumeMounts, mount)
			}
			deploymentContainer.VolumeMounts = containerVolumeMounts
		}

		deploymentContainers = append(deploymentContainers, deploymentContainer)
	}

	updateReq.Containers = deploymentContainers

	deployment, err := r.client.ContainerDeployments.UpdateDeployment(ctx, data.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update container deployment, got error: %s", err))
		return
	}

	// Fetch scaling configuration after update
	scalingConfig, err := r.client.ContainerDeployments.GetDeploymentScaling(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scaling configuration, got error: %s", err))
		return
	}

	planContainers := data.Containers
	r.flattenDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	r.flattenScalingToModel(ctx, scalingConfig, &data, &resp.Diagnostics)
	r.mergeContainersFromPlan(ctx, planContainers, &data, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ContainerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Initiate deletion (ignore timeout errors as we'll poll instead)
	err := r.client.ContainerDeployments.DeleteDeployment(ctx, data.Name.ValueString(), 60000)
	if err != nil && !isTimeoutError(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete container deployment, got error: %s", err))
		return
	}

	// Poll until deployment is gone (404) with 5 minute timeout
	if err := r.waitForDeletionComplete(ctx, data.Name.ValueString(), 300); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Timeout waiting for container deployment deletion: %s", err))
		return
	}
}

func (r *ContainerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "504") || strings.Contains(strings.ToLower(errStr), "timeout")
}

func (r *ContainerResource) waitForDeletionComplete(ctx context.Context, deploymentName string, timeoutSeconds int) error {
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	for time.Now().Before(deadline) {
		// Check if context was cancelled
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		// Try to get the deployment
		_, err := r.client.ContainerDeployments.GetDeploymentByName(ctx, deploymentName)
		if err != nil {
			// Check if it's a 404 error (deployment not found = successfully deleted)
			errStr := err.Error()
			if strings.Contains(errStr, "404") || strings.Contains(strings.ToLower(errStr), "not found") {
				return nil
			}
			// For other errors, continue polling (deployment might be in transition)
		}

		// Wait 10 seconds before trying again
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout after %d seconds waiting for deployment deletion", timeoutSeconds)
}

func (r *ContainerResource) flattenDeploymentToModel(ctx context.Context, deployment *verda.ContainerDeployment, data *ContainerResourceModel, diagnostics *diag.Diagnostics) {
	data.Name = types.StringValue(deployment.Name)
	data.IsSpot = types.BoolValue(deployment.IsSpot)
	data.EndpointBaseURL = types.StringValue(deployment.EndpointBaseURL)
	data.CreatedAt = types.StringValue(deployment.CreatedAt.Format("2006-01-02T15:04:05Z"))

	// Flatten compute
	if deployment.Compute != nil {
		computeObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"name": types.StringType,
				"size": types.Int64Type,
			},
			map[string]attr.Value{
				"name": types.StringValue(deployment.Compute.Name),
				"size": types.Int64Value(int64(deployment.Compute.Size)),
			},
		)
		diagnostics.Append(diags...)
		data.Compute = computeObj
	}

	// Flatten container registry settings
	if deployment.ContainerRegistrySettings != nil {
		registryAttrTypes := map[string]attr.Type{
			"is_private":  types.StringType,
			"credentials": types.StringType,
		}

		registryAttrValues := map[string]attr.Value{
			"is_private": types.StringValue(fmt.Sprintf("%t", deployment.ContainerRegistrySettings.IsPrivate)),
		}

		if deployment.ContainerRegistrySettings.Credentials != nil {
			registryAttrValues["credentials"] = types.StringValue(deployment.ContainerRegistrySettings.Credentials.Name)
		} else {
			registryAttrValues["credentials"] = types.StringNull()
		}

		registryObj, diags := types.ObjectValue(registryAttrTypes, registryAttrValues)
		diagnostics.Append(diags...)
		data.ContainerRegistrySettings = registryObj
	}

	// Flatten containers - only include user-specified data, filter out API-added mounts
	r.flattenContainersToModel(ctx, deployment.Containers, data, diagnostics)
}

// mergeContainersFromPlan merges plan/state container data with API response
// This preserves fields that the API doesn't return (like volume_id for non-shared volumes)
func (r *ContainerResource) mergeContainersFromPlan(ctx context.Context, planContainers types.List, data *ContainerResourceModel, diagnostics *diag.Diagnostics) {
	if planContainers.IsNull() || planContainers.IsUnknown() {
		return
	}

	var planContainersList []ContainerModel
	diags := planContainers.ElementsAs(ctx, &planContainersList, false)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	var apiContainersList []ContainerModel
	if !data.Containers.IsNull() && !data.Containers.IsUnknown() {
		diags = data.Containers.ElementsAs(ctx, &apiContainersList, false)
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return
		}
	}

	// If API didn't return containers or returned fewer containers, use plan as-is
	if len(apiContainersList) == 0 || len(apiContainersList) != len(planContainersList) {
		data.Containers = planContainers
		return
	}

	// Merge each container: use API data where available, fill in from plan where not
	var mergedContainers []attr.Value
	for i := range planContainersList {
		if i >= len(apiContainersList) {
			break
		}

		planContainer := planContainersList[i]
		apiContainer := apiContainersList[i]

		// For volume_mounts, merge carefully
		// Use plan volume_mounts since API doesn't return all fields (like volume_id for non-shared)
		mergedContainer := apiContainer
		mergedContainer.VolumeMounts = planContainer.VolumeMounts

		// Also preserve entrypoint_overrides from plan if API didn't return it
		if (apiContainer.EntrypointOverrides.IsNull() || apiContainer.EntrypointOverrides.IsUnknown()) &&
			!planContainer.EntrypointOverrides.IsNull() {
			mergedContainer.EntrypointOverrides = planContainer.EntrypointOverrides
		}

		// Convert back to attr.Value
		containerAttrTypes := map[string]attr.Type{
			"image":        types.StringType,
			"exposed_port": types.Int64Type,
			"healthcheck": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"enabled": types.StringType,
					"port":    types.StringType,
					"path":    types.StringType,
				},
			},
			"entrypoint_overrides": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"enabled":    types.BoolType,
					"entrypoint": types.ListType{ElemType: types.StringType},
					"cmd":        types.ListType{ElemType: types.StringType},
				},
			},
			"env": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":                         types.StringType,
						"name":                         types.StringType,
						"value_or_reference_to_secret": types.StringType,
					},
				},
			},
			"volume_mounts": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":        types.StringType,
						"mount_path":  types.StringType,
						"secret_name": types.StringType,
						"size_in_mb":  types.Int64Type,
						"volume_id":   types.StringType,
					},
				},
			},
		}

		containerAttrValues := map[string]attr.Value{
			"image":                mergedContainer.Image,
			"exposed_port":         mergedContainer.ExposedPort,
			"healthcheck":          mergedContainer.Healthcheck,
			"entrypoint_overrides": mergedContainer.EntrypointOverrides,
			"env":                  mergedContainer.Env,
			"volume_mounts":        mergedContainer.VolumeMounts,
		}

		containerObj, diags := types.ObjectValue(containerAttrTypes, containerAttrValues)
		diagnostics.Append(diags...)
		mergedContainers = append(mergedContainers, containerObj)
	}

	// Create the merged containers list
	containersList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"image":        types.StringType,
				"exposed_port": types.Int64Type,
				"healthcheck": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"enabled": types.StringType,
						"port":    types.StringType,
						"path":    types.StringType,
					},
				},
				"entrypoint_overrides": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"enabled":    types.BoolType,
						"entrypoint": types.ListType{ElemType: types.StringType},
						"cmd":        types.ListType{ElemType: types.StringType},
					},
				},
				"env": types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"type":                         types.StringType,
							"name":                         types.StringType,
							"value_or_reference_to_secret": types.StringType,
						},
					},
				},
				"volume_mounts": types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"type":        types.StringType,
							"mount_path":  types.StringType,
							"secret_name": types.StringType,
							"size_in_mb":  types.Int64Type,
							"volume_id":   types.StringType,
						},
					},
				},
			},
		},
		mergedContainers,
	)
	diagnostics.Append(diags...)
	data.Containers = containersList
}

func (r *ContainerResource) flattenContainersToModel(ctx context.Context, containers []verda.DeploymentContainer, data *ContainerResourceModel, diagnostics *diag.Diagnostics) {
	if len(containers) == 0 {
		return
	}

	var containerElements []attr.Value

	for _, container := range containers {
		// Build healthcheck object if present and enabled
		// Only include healthcheck in state if it's actually enabled
		var healthcheckObj types.Object
		if container.Healthcheck != nil && container.Healthcheck.Enabled {
			healthcheckAttrTypes := map[string]attr.Type{
				"enabled": types.StringType,
				"port":    types.StringType,
				"path":    types.StringType,
			}

			healthcheckAttrValues := map[string]attr.Value{
				"enabled": types.StringValue(fmt.Sprintf("%t", container.Healthcheck.Enabled)),
			}

			if container.Healthcheck.Port != 0 {
				healthcheckAttrValues["port"] = types.StringValue(fmt.Sprintf("%d", container.Healthcheck.Port))
			} else {
				healthcheckAttrValues["port"] = types.StringNull()
			}

			if container.Healthcheck.Path != "" {
				healthcheckAttrValues["path"] = types.StringValue(container.Healthcheck.Path)
			} else {
				healthcheckAttrValues["path"] = types.StringNull()
			}

			hcObj, diags := types.ObjectValue(healthcheckAttrTypes, healthcheckAttrValues)
			diagnostics.Append(diags...)
			healthcheckObj = hcObj
		} else {
			healthcheckObj = types.ObjectNull(map[string]attr.Type{
				"enabled": types.StringType,
				"port":    types.StringType,
				"path":    types.StringType,
			})
		}

		// Build entrypoint overrides object if present
		var entrypointOverridesObj types.Object
		if container.EntrypointOverrides != nil && container.EntrypointOverrides.Enabled {
			entrypointList, diags := types.ListValueFrom(ctx, types.StringType, container.EntrypointOverrides.Entrypoint)
			diagnostics.Append(diags...)

			cmdList, diags := types.ListValueFrom(ctx, types.StringType, container.EntrypointOverrides.Cmd)
			diagnostics.Append(diags...)

			entrypointOverridesAttrTypes := map[string]attr.Type{
				"enabled":    types.BoolType,
				"entrypoint": types.ListType{ElemType: types.StringType},
				"cmd":        types.ListType{ElemType: types.StringType},
			}

			entrypointOverridesAttrValues := map[string]attr.Value{
				"enabled":    types.BoolValue(container.EntrypointOverrides.Enabled),
				"entrypoint": entrypointList,
				"cmd":        cmdList,
			}

			epObj, diags := types.ObjectValue(entrypointOverridesAttrTypes, entrypointOverridesAttrValues)
			diagnostics.Append(diags...)
			entrypointOverridesObj = epObj
		} else {
			entrypointOverridesObj = types.ObjectNull(map[string]attr.Type{
				"enabled":    types.BoolType,
				"entrypoint": types.ListType{ElemType: types.StringType},
				"cmd":        types.ListType{ElemType: types.StringType},
			})
		}

		// Build env vars list
		var envList types.List
		if len(container.Env) > 0 {
			var envElements []attr.Value
			for _, envVar := range container.Env {
				envAttrTypes := map[string]attr.Type{
					"type":                         types.StringType,
					"name":                         types.StringType,
					"value_or_reference_to_secret": types.StringType,
				}

				envAttrValues := map[string]attr.Value{
					"type":                         types.StringValue(envVar.Type),
					"name":                         types.StringValue(envVar.Name),
					"value_or_reference_to_secret": types.StringValue(envVar.ValueOrReferenceToSecret),
				}

				envObj, diags := types.ObjectValue(envAttrTypes, envAttrValues)
				diagnostics.Append(diags...)
				envElements = append(envElements, envObj)
			}

			envListVal, diags := types.ListValue(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":                         types.StringType,
						"name":                         types.StringType,
						"value_or_reference_to_secret": types.StringType,
					},
				},
				envElements,
			)
			diagnostics.Append(diags...)
			envList = envListVal
		} else {
			envList = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":                         types.StringType,
					"name":                         types.StringType,
					"value_or_reference_to_secret": types.StringType,
				},
			})
		}

		// Build volume mounts list
		// Note: We don't filter API-added mounts here because the merge function
		// preserves volume_mounts from plan/state, which already has the correct data
		var volumeMountsList types.List
		if len(container.VolumeMounts) > 0 {
			var volumeMountElements []attr.Value
			for _, mount := range container.VolumeMounts {
				volumeMountAttrTypes := map[string]attr.Type{
					"type":        types.StringType,
					"mount_path":  types.StringType,
					"secret_name": types.StringType,
					"size_in_mb":  types.Int64Type,
					"volume_id":   types.StringType,
				}

				volumeMountAttrValues := map[string]attr.Value{
					"type":       types.StringValue(mount.Type),
					"mount_path": types.StringValue(mount.MountPath),
				}

				if mount.SecretName != "" {
					volumeMountAttrValues["secret_name"] = types.StringValue(mount.SecretName)
				} else {
					volumeMountAttrValues["secret_name"] = types.StringNull()
				}

				if mount.SizeInMB != 0 {
					volumeMountAttrValues["size_in_mb"] = types.Int64Value(int64(mount.SizeInMB))
				} else {
					volumeMountAttrValues["size_in_mb"] = types.Int64Null()
				}

				if mount.VolumeID != "" {
					volumeMountAttrValues["volume_id"] = types.StringValue(mount.VolumeID)
				} else {
					volumeMountAttrValues["volume_id"] = types.StringNull()
				}

				volumeMountObj, diags := types.ObjectValue(volumeMountAttrTypes, volumeMountAttrValues)
				diagnostics.Append(diags...)
				volumeMountElements = append(volumeMountElements, volumeMountObj)
			}

			volumeMountsListVal, diags := types.ListValue(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":        types.StringType,
						"mount_path":  types.StringType,
						"secret_name": types.StringType,
						"size_in_mb":  types.Int64Type,
						"volume_id":   types.StringType,
					},
				},
				volumeMountElements,
			)
			diagnostics.Append(diags...)
			volumeMountsList = volumeMountsListVal
		} else {
			volumeMountsList = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":        types.StringType,
					"mount_path":  types.StringType,
					"secret_name": types.StringType,
					"size_in_mb":  types.Int64Type,
					"volume_id":   types.StringType,
				},
			})
		}

		// Build the container object
		containerAttrTypes := map[string]attr.Type{
			"image":        types.StringType,
			"exposed_port": types.Int64Type,
			"healthcheck": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"enabled": types.StringType,
					"port":    types.StringType,
					"path":    types.StringType,
				},
			},
			"entrypoint_overrides": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"enabled":    types.BoolType,
					"entrypoint": types.ListType{ElemType: types.StringType},
					"cmd":        types.ListType{ElemType: types.StringType},
				},
			},
			"env": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":                         types.StringType,
						"name":                         types.StringType,
						"value_or_reference_to_secret": types.StringType,
					},
				},
			},
			"volume_mounts": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":        types.StringType,
						"mount_path":  types.StringType,
						"secret_name": types.StringType,
						"size_in_mb":  types.Int64Type,
						"volume_id":   types.StringType,
					},
				},
			},
		}

		containerAttrValues := map[string]attr.Value{
			"image":                types.StringValue(container.Image.Image),
			"exposed_port":         types.Int64Value(int64(container.ExposedPort)),
			"healthcheck":          healthcheckObj,
			"entrypoint_overrides": entrypointOverridesObj,
			"env":                  envList,
			"volume_mounts":        volumeMountsList,
		}

		containerObj, diags := types.ObjectValue(containerAttrTypes, containerAttrValues)
		diagnostics.Append(diags...)
		containerElements = append(containerElements, containerObj)
	}

	// Create the containers list
	containersList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"image":        types.StringType,
				"exposed_port": types.Int64Type,
				"healthcheck": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"enabled": types.StringType,
						"port":    types.StringType,
						"path":    types.StringType,
					},
				},
				"entrypoint_overrides": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"enabled":    types.BoolType,
						"entrypoint": types.ListType{ElemType: types.StringType},
						"cmd":        types.ListType{ElemType: types.StringType},
					},
				},
				"env": types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"type":                         types.StringType,
							"name":                         types.StringType,
							"value_or_reference_to_secret": types.StringType,
						},
					},
				},
				"volume_mounts": types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"type":        types.StringType,
							"mount_path":  types.StringType,
							"secret_name": types.StringType,
							"size_in_mb":  types.Int64Type,
							"volume_id":   types.StringType,
						},
					},
				},
			},
		},
		containerElements,
	)
	diagnostics.Append(diags...)
	data.Containers = containersList
}

func (r *ContainerResource) flattenScalingToModel(ctx context.Context, scalingConfig *verda.ContainerScalingOptions, data *ContainerResourceModel, diagnostics *diag.Diagnostics) {
	scaleDownPolicyObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"delay_seconds": types.Int64Type,
		},
		map[string]attr.Value{
			"delay_seconds": types.Int64Value(int64(scalingConfig.ScaleDownPolicy.DelaySeconds)),
		},
	)
	diagnostics.Append(diags...)

	scaleUpPolicyObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"delay_seconds": types.Int64Type,
		},
		map[string]attr.Value{
			"delay_seconds": types.Int64Value(int64(scalingConfig.ScaleUpPolicy.DelaySeconds)),
		},
	)
	diagnostics.Append(diags...)

	queueLoadObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"threshold": types.Float64Type,
		},
		map[string]attr.Value{
			"threshold": types.Float64Value(scalingConfig.ScalingTriggers.QueueLoad.Threshold),
		},
	)
	diagnostics.Append(diags...)

	scalingObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"min_replica_count":               types.Int64Type,
			"max_replica_count":               types.Int64Type,
			"queue_message_ttl_seconds":       types.Int64Type,
			"deadline_seconds":                types.Int64Type,
			"concurrent_requests_per_replica": types.Int64Type,
			"scale_down_policy": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"delay_seconds": types.Int64Type,
				},
			},
			"scale_up_policy": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"delay_seconds": types.Int64Type,
				},
			},
			"queue_load": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"threshold": types.Float64Type,
				},
			},
		},
		map[string]attr.Value{
			"min_replica_count":               types.Int64Value(int64(scalingConfig.MinReplicaCount)),
			"max_replica_count":               types.Int64Value(int64(scalingConfig.MaxReplicaCount)),
			"queue_message_ttl_seconds":       types.Int64Value(int64(scalingConfig.QueueMessageTTLSeconds)),
			"deadline_seconds":                types.Int64Value(int64(scalingConfig.QueueMessageTTLSeconds)),
			"concurrent_requests_per_replica": types.Int64Value(int64(scalingConfig.ConcurrentRequestsPerReplica)),
			"scale_down_policy":               scaleDownPolicyObj,
			"scale_up_policy":                 scaleUpPolicyObj,
			"queue_load":                      queueLoadObj,
		},
	)
	diagnostics.Append(diags...)
	data.Scaling = scalingObj
}

// boolDefaultModifier is defined in resource_instance.go but we need it here too
// volumeMountValidator is now defined in validators.go and shared across resources
