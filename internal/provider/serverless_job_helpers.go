package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var (
	jobComputeAttrTypes = map[string]attr.Type{
		"name": types.StringType,
		"size": types.Int64Type,
	}
	jobScalingAttrTypes = map[string]attr.Type{
		"max_replica_count":         types.Int64Type,
		"queue_message_ttl_seconds": types.Int64Type,
		"deadline_seconds":          types.Int64Type,
	}
	jobRegistryAttrTypes = map[string]attr.Type{
		"is_private":  types.StringType,
		"credentials": types.StringType,
	}
	jobHealthcheckAttrTypes = map[string]attr.Type{
		"enabled": types.StringType,
		"port":    types.StringType,
		"path":    types.StringType,
	}
	jobEntrypointOverridesAttrTypes = map[string]attr.Type{
		"enabled":    types.BoolType,
		"entrypoint": types.ListType{ElemType: types.StringType},
		"cmd":        types.ListType{ElemType: types.StringType},
	}
	jobEnvVarAttrTypes = map[string]attr.Type{
		"type":                         types.StringType,
		"name":                         types.StringType,
		"value_or_reference_to_secret": types.StringType,
	}
	jobEnvVarObjectType     = types.ObjectType{AttrTypes: jobEnvVarAttrTypes}
	jobVolumeMountAttrTypes = map[string]attr.Type{
		"type":        types.StringType,
		"mount_path":  types.StringType,
		"secret_name": types.StringType,
		"size_in_mb":  types.Int64Type,
		"volume_id":   types.StringType,
	}
	jobVolumeMountObjectType = types.ObjectType{AttrTypes: jobVolumeMountAttrTypes}
	jobContainerAttrTypes    = map[string]attr.Type{
		"image":                types.StringType,
		"exposed_port":         types.Int64Type,
		"healthcheck":          types.ObjectType{AttrTypes: jobHealthcheckAttrTypes},
		"entrypoint_overrides": types.ObjectType{AttrTypes: jobEntrypointOverridesAttrTypes},
		"env":                  types.ListType{ElemType: jobEnvVarObjectType},
		"volume_mounts":        types.ListType{ElemType: jobVolumeMountObjectType},
	}
	jobContainerObjectType = types.ObjectType{AttrTypes: jobContainerAttrTypes}
)

type serverlessJobRequestParts struct {
	compute                   *verda.ContainerCompute
	scaling                   *verda.JobScalingOptions
	containerRegistrySettings *verda.ContainerRegistrySettings
	hasContainerRegistry      bool
	containers                []verda.CreateDeploymentContainer
}

func buildServerlessJobRequestParts(ctx context.Context, data ServerlessJobResourceModel, diagnostics *diag.Diagnostics) *serverlessJobRequestParts {
	var compute ComputeModel
	diagnostics.Append(data.Compute.As(ctx, &compute, basetypes.ObjectAsOptions{})...)
	if diagnostics.HasError() {
		return nil
	}

	var scaling JobScalingModel
	diagnostics.Append(data.Scaling.As(ctx, &scaling, basetypes.ObjectAsOptions{})...)
	if diagnostics.HasError() {
		return nil
	}

	parts := &serverlessJobRequestParts{
		compute: &verda.ContainerCompute{
			Name: compute.Name.ValueString(),
			Size: int(compute.Size.ValueInt64()),
		},
		scaling: &verda.JobScalingOptions{
			MaxReplicaCount:        int(scaling.MaxReplicaCount.ValueInt64()),
			QueueMessageTTLSeconds: int(scaling.QueueMessageTTLSeconds.ValueInt64()),
			DeadlineSeconds:        int(scaling.DeadlineSeconds.ValueInt64()),
		},
	}

	if !data.ContainerRegistrySettings.IsNull() && !data.ContainerRegistrySettings.IsUnknown() {
		var registrySettings RegistrySettingsModel
		diagnostics.Append(data.ContainerRegistrySettings.As(ctx, &registrySettings, basetypes.ObjectAsOptions{})...)
		if diagnostics.HasError() {
			return nil
		}

		parts.hasContainerRegistry = true
		parts.containerRegistrySettings = &verda.ContainerRegistrySettings{
			IsPrivate: registrySettings.IsPrivate.ValueString() == "true",
		}

		if !registrySettings.Credentials.IsNull() && registrySettings.Credentials.ValueString() != "" {
			parts.containerRegistrySettings.Credentials = &verda.RegistryCredentialsRef{
				Name: registrySettings.Credentials.ValueString(),
			}
		}
	}

	var containers []ContainerModel
	diagnostics.Append(data.Containers.ElementsAs(ctx, &containers, false)...)
	if diagnostics.HasError() {
		return nil
	}

	parts.containers = make([]verda.CreateDeploymentContainer, 0, len(containers))
	for _, container := range containers {
		deploymentContainer := verda.CreateDeploymentContainer{
			Image:       container.Image.ValueString(),
			ExposedPort: int(container.ExposedPort.ValueInt64()),
		}

		if !container.Healthcheck.IsNull() {
			var healthcheck HealthcheckModel
			diagnostics.Append(container.Healthcheck.As(ctx, &healthcheck, basetypes.ObjectAsOptions{})...)
			if diagnostics.HasError() {
				return nil
			}

			hc := &verda.ContainerHealthcheck{
				Enabled: healthcheck.Enabled.ValueString() == "true",
			}

			if !healthcheck.Port.IsNull() && healthcheck.Port.ValueString() != "" {
				var port int
				if _, err := fmt.Sscanf(healthcheck.Port.ValueString(), "%d", &port); err == nil {
					hc.Port = port
				}
			}

			if !healthcheck.Path.IsNull() {
				hc.Path = healthcheck.Path.ValueString()
			}

			deploymentContainer.Healthcheck = hc
		}

		if !container.EntrypointOverrides.IsNull() && !container.EntrypointOverrides.IsUnknown() {
			var entrypointOverrides EntrypointOverridesModel
			diagnostics.Append(container.EntrypointOverrides.As(ctx, &entrypointOverrides, basetypes.ObjectAsOptions{})...)
			if diagnostics.HasError() {
				return nil
			}

			overrides := &verda.ContainerEntrypointOverrides{
				Enabled: entrypointOverrides.Enabled.ValueBool(),
			}

			if !entrypointOverrides.Entrypoint.IsNull() && !entrypointOverrides.Entrypoint.IsUnknown() {
				var entrypoint []string
				diagnostics.Append(entrypointOverrides.Entrypoint.ElementsAs(ctx, &entrypoint, false)...)
				if diagnostics.HasError() {
					return nil
				}
				overrides.Entrypoint = entrypoint
			}

			if !entrypointOverrides.Cmd.IsNull() && !entrypointOverrides.Cmd.IsUnknown() {
				var cmd []string
				diagnostics.Append(entrypointOverrides.Cmd.ElementsAs(ctx, &cmd, false)...)
				if diagnostics.HasError() {
					return nil
				}
				overrides.Cmd = cmd
			}

			deploymentContainer.EntrypointOverrides = overrides
		}

		if !container.Env.IsNull() {
			var envVars []EnvVarModel
			diagnostics.Append(container.Env.ElementsAs(ctx, &envVars, false)...)
			if diagnostics.HasError() {
				return nil
			}

			deploymentContainer.Env = make([]verda.ContainerEnvVar, 0, len(envVars))
			for _, envVar := range envVars {
				deploymentContainer.Env = append(deploymentContainer.Env, verda.ContainerEnvVar{
					Type:                     envVar.Type.ValueString(),
					Name:                     envVar.Name.ValueString(),
					ValueOrReferenceToSecret: envVar.ValueOrReferenceToSecret.ValueString(),
				})
			}
		}

		if !container.VolumeMounts.IsNull() {
			var volumeMounts []VolumeMountModel
			diagnostics.Append(container.VolumeMounts.ElementsAs(ctx, &volumeMounts, false)...)
			if diagnostics.HasError() {
				return nil
			}

			deploymentContainer.VolumeMounts = make([]verda.ContainerVolumeMount, 0, len(volumeMounts))
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

				deploymentContainer.VolumeMounts = append(deploymentContainer.VolumeMounts, mount)
			}
		}

		parts.containers = append(parts.containers, deploymentContainer)
	}

	return parts
}

func buildServerlessJobCreateRequest(ctx context.Context, data ServerlessJobResourceModel, diagnostics *diag.Diagnostics) *verda.CreateJobDeploymentRequest {
	parts := buildServerlessJobRequestParts(ctx, data, diagnostics)
	if diagnostics.HasError() || parts == nil {
		return nil
	}

	req := &verda.CreateJobDeploymentRequest{
		Name:       data.Name.ValueString(),
		Compute:    parts.compute,
		Scaling:    parts.scaling,
		Containers: parts.containers,
	}

	if parts.hasContainerRegistry {
		req.ContainerRegistrySettings = parts.containerRegistrySettings
	} else {
		req.ContainerRegistrySettings = &verda.ContainerRegistrySettings{IsPrivate: false}
	}

	return req
}

func buildServerlessJobUpdateRequest(ctx context.Context, data ServerlessJobResourceModel, diagnostics *diag.Diagnostics) *verda.UpdateJobDeploymentRequest {
	parts := buildServerlessJobRequestParts(ctx, data, diagnostics)
	if diagnostics.HasError() || parts == nil {
		return nil
	}

	req := &verda.UpdateJobDeploymentRequest{
		Compute:    parts.compute,
		Scaling:    parts.scaling,
		Containers: parts.containers,
	}

	if parts.hasContainerRegistry {
		req.ContainerRegistrySettings = parts.containerRegistrySettings
	}

	return req
}

func flattenJobDeploymentToModel(ctx context.Context, deployment *verda.JobDeployment, data *ServerlessJobResourceModel, diagnostics *diag.Diagnostics) {
	data.Name = types.StringValue(deployment.Name)
	data.EndpointBaseURL = types.StringValue(deployment.EndpointBaseURL)
	data.CreatedAt = types.StringValue(deployment.CreatedAt.Format("2006-01-02T15:04:05Z"))

	if deployment.Compute != nil {
		computeObj, diags := types.ObjectValue(
			jobComputeAttrTypes,
			map[string]attr.Value{
				"name": types.StringValue(deployment.Compute.Name),
				"size": types.Int64Value(int64(deployment.Compute.Size)),
			},
		)
		diagnostics.Append(diags...)
		data.Compute = computeObj
	} else {
		data.Compute = types.ObjectNull(jobComputeAttrTypes)
	}

	if deployment.Scaling != nil {
		scalingObj, diags := types.ObjectValue(
			jobScalingAttrTypes,
			map[string]attr.Value{
				"max_replica_count":         types.Int64Value(int64(deployment.Scaling.MaxReplicaCount)),
				"queue_message_ttl_seconds": types.Int64Value(int64(deployment.Scaling.QueueMessageTTLSeconds)),
				"deadline_seconds":          types.Int64Value(int64(deployment.Scaling.DeadlineSeconds)),
			},
		)
		diagnostics.Append(diags...)
		data.Scaling = scalingObj
	} else {
		data.Scaling = types.ObjectNull(jobScalingAttrTypes)
	}

	if deployment.ContainerRegistrySettings != nil {
		registryValues := map[string]attr.Value{
			"is_private":  types.StringValue(fmt.Sprintf("%t", deployment.ContainerRegistrySettings.IsPrivate)),
			"credentials": types.StringNull(),
		}
		if deployment.ContainerRegistrySettings.Credentials != nil {
			registryValues["credentials"] = types.StringValue(deployment.ContainerRegistrySettings.Credentials.Name)
		}

		registryObj, diags := types.ObjectValue(jobRegistryAttrTypes, registryValues)
		diagnostics.Append(diags...)
		data.ContainerRegistrySettings = registryObj
	} else {
		data.ContainerRegistrySettings = types.ObjectNull(jobRegistryAttrTypes)
	}

	flattenJobContainersToModel(ctx, deployment.Containers, data, diagnostics)
}

func flattenJobContainersToModel(ctx context.Context, containers []verda.DeploymentContainer, data *ServerlessJobResourceModel, diagnostics *diag.Diagnostics) {
	if len(containers) == 0 {
		data.Containers = types.ListNull(jobContainerObjectType)
		return
	}

	containerElements := make([]attr.Value, 0, len(containers))
	for _, container := range containers {
		healthcheckObj := types.ObjectNull(jobHealthcheckAttrTypes)
		if container.Healthcheck != nil && container.Healthcheck.Enabled {
			healthcheckValues := map[string]attr.Value{
				"enabled": types.StringValue(fmt.Sprintf("%t", container.Healthcheck.Enabled)),
				"port":    types.StringNull(),
				"path":    types.StringNull(),
			}
			if container.Healthcheck.Port != 0 {
				healthcheckValues["port"] = types.StringValue(fmt.Sprintf("%d", container.Healthcheck.Port))
			}
			if container.Healthcheck.Path != "" {
				healthcheckValues["path"] = types.StringValue(container.Healthcheck.Path)
			}

			hcObj, diags := types.ObjectValue(jobHealthcheckAttrTypes, healthcheckValues)
			diagnostics.Append(diags...)
			healthcheckObj = hcObj
		}

		entrypointOverridesObj := types.ObjectNull(jobEntrypointOverridesAttrTypes)
		if container.EntrypointOverrides != nil && container.EntrypointOverrides.Enabled {
			entrypointList, diags := types.ListValueFrom(ctx, types.StringType, container.EntrypointOverrides.Entrypoint)
			diagnostics.Append(diags...)
			cmdList, diags := types.ListValueFrom(ctx, types.StringType, container.EntrypointOverrides.Cmd)
			diagnostics.Append(diags...)

			epObj, diags := types.ObjectValue(
				jobEntrypointOverridesAttrTypes,
				map[string]attr.Value{
					"enabled":    types.BoolValue(container.EntrypointOverrides.Enabled),
					"entrypoint": entrypointList,
					"cmd":        cmdList,
				},
			)
			diagnostics.Append(diags...)
			entrypointOverridesObj = epObj
		}

		envList := types.ListNull(jobEnvVarObjectType)
		if len(container.Env) > 0 {
			envElements := make([]attr.Value, 0, len(container.Env))
			for _, envVar := range container.Env {
				envObj, diags := types.ObjectValue(
					jobEnvVarAttrTypes,
					map[string]attr.Value{
						"type":                         types.StringValue(envVar.Type),
						"name":                         types.StringValue(envVar.Name),
						"value_or_reference_to_secret": types.StringValue(envVar.ValueOrReferenceToSecret),
					},
				)
				diagnostics.Append(diags...)
				envElements = append(envElements, envObj)
			}

			envListVal, diags := types.ListValue(jobEnvVarObjectType, envElements)
			diagnostics.Append(diags...)
			envList = envListVal
		}

		volumeMountsList := types.ListNull(jobVolumeMountObjectType)
		if len(container.VolumeMounts) > 0 {
			volumeMountElements := make([]attr.Value, 0, len(container.VolumeMounts))
			for _, mount := range container.VolumeMounts {
				volumeMountValues := map[string]attr.Value{
					"type":        types.StringValue(mount.Type),
					"mount_path":  types.StringValue(mount.MountPath),
					"secret_name": types.StringNull(),
					"size_in_mb":  types.Int64Null(),
					"volume_id":   types.StringNull(),
				}

				if mount.SecretName != "" {
					volumeMountValues["secret_name"] = types.StringValue(mount.SecretName)
				}
				if mount.SizeInMB != 0 {
					volumeMountValues["size_in_mb"] = types.Int64Value(int64(mount.SizeInMB))
				}
				if mount.VolumeID != "" {
					volumeMountValues["volume_id"] = types.StringValue(mount.VolumeID)
				}

				volumeMountObj, diags := types.ObjectValue(jobVolumeMountAttrTypes, volumeMountValues)
				diagnostics.Append(diags...)
				volumeMountElements = append(volumeMountElements, volumeMountObj)
			}

			volumeMountsListVal, diags := types.ListValue(jobVolumeMountObjectType, volumeMountElements)
			diagnostics.Append(diags...)
			volumeMountsList = volumeMountsListVal
		}

		containerObj, diags := types.ObjectValue(
			jobContainerAttrTypes,
			map[string]attr.Value{
				"image":                types.StringValue(container.Image.Image),
				"exposed_port":         types.Int64Value(int64(container.ExposedPort)),
				"healthcheck":          healthcheckObj,
				"entrypoint_overrides": entrypointOverridesObj,
				"env":                  envList,
				"volume_mounts":        volumeMountsList,
			},
		)
		diagnostics.Append(diags...)
		containerElements = append(containerElements, containerObj)
	}

	containersList, diags := types.ListValue(jobContainerObjectType, containerElements)
	diagnostics.Append(diags...)
	data.Containers = containersList
}

func mergeJobContainersFromPlan(ctx context.Context, planContainers types.List, data *ServerlessJobResourceModel, diagnostics *diag.Diagnostics) {
	if planContainers.IsNull() || planContainers.IsUnknown() {
		return
	}

	var planContainersList []ContainerModel
	diagnostics.Append(planContainers.ElementsAs(ctx, &planContainersList, false)...)
	if diagnostics.HasError() {
		return
	}

	var apiContainersList []ContainerModel
	if !data.Containers.IsNull() && !data.Containers.IsUnknown() {
		diagnostics.Append(data.Containers.ElementsAs(ctx, &apiContainersList, false)...)
		if diagnostics.HasError() {
			return
		}
	}

	if len(apiContainersList) == 0 || len(apiContainersList) != len(planContainersList) {
		data.Containers = planContainers
		return
	}

	mergedContainers := make([]attr.Value, 0, len(planContainersList))
	for i := range planContainersList {
		if i >= len(apiContainersList) {
			break
		}

		planContainer := planContainersList[i]
		apiContainer := apiContainersList[i]
		mergedContainer := apiContainer
		mergedContainer.VolumeMounts = planContainer.VolumeMounts

		if (apiContainer.EntrypointOverrides.IsNull() || apiContainer.EntrypointOverrides.IsUnknown()) &&
			!planContainer.EntrypointOverrides.IsNull() {
			mergedContainer.EntrypointOverrides = planContainer.EntrypointOverrides
		}

		containerObj, diags := types.ObjectValue(
			jobContainerAttrTypes,
			map[string]attr.Value{
				"image":                mergedContainer.Image,
				"exposed_port":         mergedContainer.ExposedPort,
				"healthcheck":          mergedContainer.Healthcheck,
				"entrypoint_overrides": mergedContainer.EntrypointOverrides,
				"env":                  mergedContainer.Env,
				"volume_mounts":        mergedContainer.VolumeMounts,
			},
		)
		diagnostics.Append(diags...)
		mergedContainers = append(mergedContainers, containerObj)
	}

	containersList, diags := types.ListValue(jobContainerObjectType, mergedContainers)
	diagnostics.Append(diags...)
	data.Containers = containersList
}

func flattenJobDeploymentShortInfos(ctx context.Context, jobs []verda.JobDeploymentShortInfo, diagnostics *diag.Diagnostics) types.List {
	jobSummaryAttrTypes := map[string]attr.Type{
		"name":       types.StringType,
		"created_at": types.StringType,
		"compute":    types.ObjectType{AttrTypes: jobComputeAttrTypes},
	}

	if len(jobs) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: jobSummaryAttrTypes})
	}

	jobElements := make([]attr.Value, 0, len(jobs))
	for _, job := range jobs {
		computeValue := types.ObjectNull(jobComputeAttrTypes)
		if job.Compute != nil {
			computeObj, diags := types.ObjectValue(
				jobComputeAttrTypes,
				map[string]attr.Value{
					"name": types.StringValue(job.Compute.Name),
					"size": types.Int64Value(int64(job.Compute.Size)),
				},
			)
			diagnostics.Append(diags...)
			computeValue = computeObj
		}

		jobObj, diags := types.ObjectValue(
			jobSummaryAttrTypes,
			map[string]attr.Value{
				"name":       types.StringValue(job.Name),
				"created_at": types.StringValue(job.CreatedAt.Format("2006-01-02T15:04:05Z")),
				"compute":    computeValue,
			},
		)
		diagnostics.Append(diags...)
		jobElements = append(jobElements, jobObj)
	}

	jobsList, diags := types.ListValue(types.ObjectType{AttrTypes: jobSummaryAttrTypes}, jobElements)
	diagnostics.Append(diags...)
	return jobsList
}
