package provider

import (
	"context"
	"fmt"

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

	createReq := &verda.CreateJobDeploymentRequest{
		Name: data.Name.ValueString(),
	}

	// Parse compute
	var compute ComputeModel
	resp.Diagnostics.Append(data.Compute.As(ctx, &compute, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq.Compute = &verda.ContainerCompute{
		Name: compute.Name.ValueString(),
		Size: int(compute.Size.ValueInt64()),
	}

	var scaling JobScalingModel
	resp.Diagnostics.Append(data.Scaling.As(ctx, &scaling, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	scalingOptions := verda.JobScalingOptions{
		MaxReplicaCount:        int(scaling.MaxReplicaCount.ValueInt64()),
		QueueMessageTTLSeconds: int(scaling.QueueMessageTTLSeconds.ValueInt64()),
		DeadlineSeconds:        int(scaling.DeadlineSeconds.ValueInt64()),
	}

	createReq.Scaling = &scalingOptions

	// Parse container registry settings if provided
	if !data.ContainerRegistrySettings.IsNull() && !data.ContainerRegistrySettings.IsUnknown() {
		var registrySettings RegistrySettingsModel
		resp.Diagnostics.Append(data.ContainerRegistrySettings.As(ctx, &registrySettings, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		isPrivate := registrySettings.IsPrivate.ValueString() == "true"
		createReq.ContainerRegistrySettings = &verda.ContainerRegistrySettings{
			IsPrivate: isPrivate,
		}

		if !registrySettings.Credentials.IsNull() && registrySettings.Credentials.ValueString() != "" {
			createReq.ContainerRegistrySettings.Credentials = &verda.RegistryCredentialsRef{
				Name: registrySettings.Credentials.ValueString(),
			}
		}
	} else {
		createReq.ContainerRegistrySettings = &verda.ContainerRegistrySettings{
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

	deployment, err := r.client.ServerlessJobs.CreateJobDeployment(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create serverless job deployment, got error: %s", err))
		return
	}

	// Flatten API response, merging with plan to preserve fields the API doesn't return
	planContainers := data.Containers
	r.flattenJobDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	r.mergeJobContainersFromPlan(ctx, planContainers, &data, &resp.Diagnostics)

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
	r.flattenJobDeploymentToModel(ctx, deployment, &data, &resp.Diagnostics)
	r.mergeJobContainersFromPlan(ctx, priorContainers, &data, &resp.Diagnostics)

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

	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Serverless job deployments cannot be updated. Please delete and recreate the resource with new values.",
	)
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

func (r *ServerlessJobResource) flattenJobDeploymentToModel(ctx context.Context, deployment *verda.JobDeployment, data *ServerlessJobResourceModel, diagnostics *diag.Diagnostics) {
	data.Name = types.StringValue(deployment.Name)
	data.EndpointBaseURL = types.StringValue(deployment.EndpointBaseURL)
	data.CreatedAt = types.StringValue(deployment.CreatedAt)

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

	// Flatten scaling
	if deployment.Scaling != nil {
		scalingObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"max_replica_count":         types.Int64Type,
				"queue_message_ttl_seconds": types.Int64Type,
				"deadline_seconds":          types.Int64Type,
			},
			map[string]attr.Value{
				"max_replica_count":         types.Int64Value(int64(deployment.Scaling.MaxReplicaCount)),
				"queue_message_ttl_seconds": types.Int64Value(int64(deployment.Scaling.QueueMessageTTLSeconds)),
				"deadline_seconds":          types.Int64Value(int64(deployment.Scaling.DeadlineSeconds)),
			},
		)
		diagnostics.Append(diags...)
		data.Scaling = scalingObj
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

	// Flatten containers
	r.flattenJobContainersToModel(ctx, deployment.Containers, data, diagnostics)
}

func (r *ServerlessJobResource) flattenJobContainersToModel(ctx context.Context, containers []verda.DeploymentContainer, data *ServerlessJobResourceModel, diagnostics *diag.Diagnostics) {
	if len(containers) == 0 {
		return
	}

	var containerElements []attr.Value

	for _, container := range containers {
		// Build healthcheck object if present and enabled
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

func (r *ServerlessJobResource) mergeJobContainersFromPlan(ctx context.Context, planContainers types.List, data *ServerlessJobResourceModel, diagnostics *diag.Diagnostics) {
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

		// Use plan volume_mounts since API doesn't return all fields
		mergedContainer := apiContainer
		mergedContainer.VolumeMounts = planContainer.VolumeMounts

		// Preserve entrypoint_overrides from plan if API didn't return it
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
