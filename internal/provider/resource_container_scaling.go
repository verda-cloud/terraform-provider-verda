package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &ContainerScalingResource{}
var _ resource.ResourceWithImportState = &ContainerScalingResource{}

func NewContainerScalingResource() resource.Resource {
	return &ContainerScalingResource{}
}

type ContainerScalingResource struct {
	client *verda.Client
}

type ContainerScalingResourceModel struct {
	DeploymentName               types.String `tfsdk:"deployment_name"`
	MinReplicaCount              types.Int64  `tfsdk:"min_replica_count"`
	MaxReplicaCount              types.Int64  `tfsdk:"max_replica_count"`
	QueueMessageTTLSeconds       types.Int64  `tfsdk:"queue_message_ttl_seconds"`
	ConcurrentRequestsPerReplica types.Int64  `tfsdk:"concurrent_requests_per_replica"`
	ScaleDownPolicy              types.Object `tfsdk:"scale_down_policy"`
	ScaleUpPolicy                types.Object `tfsdk:"scale_up_policy"`
	QueueLoad                    types.Object `tfsdk:"queue_load"`
	CPUUtilization               types.Object `tfsdk:"cpu_utilization"`
	GPUUtilization               types.Object `tfsdk:"gpu_utilization"`
}

type UtilizationTriggerModel struct {
	Enabled   types.Bool  `tfsdk:"enabled"`
	Threshold types.Int64 `tfsdk:"threshold"`
}

func (r *ContainerScalingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_scaling"
}

func (r *ContainerScalingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	utilizationSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{Required: true},
			"threshold": schema.Int64Attribute{
				Required: true,
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages scaling configuration for a container deployment.",
		Attributes: map[string]schema.Attribute{
			"deployment_name": schema.StringAttribute{
				MarkdownDescription: "Deployment name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"min_replica_count": schema.Int64Attribute{
				Required: true,
			},
			"max_replica_count": schema.Int64Attribute{
				Required: true,
			},
			"queue_message_ttl_seconds": schema.Int64Attribute{
				Required: true,
			},
			"concurrent_requests_per_replica": schema.Int64Attribute{
				Required: true,
			},
			"scale_down_policy": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"delay_seconds": schema.Int64Attribute{Required: true},
				},
			},
			"scale_up_policy": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"delay_seconds": schema.Int64Attribute{Required: true},
				},
			},
			"queue_load": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"threshold": schema.Float64Attribute{Required: true},
				},
			},
			"cpu_utilization": utilizationSchema,
			"gpu_utilization": utilizationSchema,
		},
	}
}

func (r *ContainerScalingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContainerScalingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerScalingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyScaling(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update container scaling, got error: %s", err))
		return
	}

	r.readScaling(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerScalingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContainerScalingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readScaling(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerScalingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ContainerScalingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyScaling(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update container scaling, got error: %s", err))
		return
	}

	r.readScaling(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerScalingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No delete operation for scaling; keep the configuration as-is.
}

func (r *ContainerScalingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("deployment_name"), req, resp)
}

func (r *ContainerScalingResource) applyScaling(ctx context.Context, data ContainerScalingResourceModel) error {
	scalingOptions := verda.ContainerScalingOptions{
		MinReplicaCount:              int(data.MinReplicaCount.ValueInt64()),
		MaxReplicaCount:              int(data.MaxReplicaCount.ValueInt64()),
		QueueMessageTTLSeconds:       int(data.QueueMessageTTLSeconds.ValueInt64()),
		ConcurrentRequestsPerReplica: int(data.ConcurrentRequestsPerReplica.ValueInt64()),
	}

	var scaleDownPolicy ScalingPolicyModel
	if diags := data.ScaleDownPolicy.As(ctx, &scaleDownPolicy, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("invalid scale_down_policy")
	}
	scalingOptions.ScaleDownPolicy = &verda.ScalingPolicy{
		DelaySeconds: int(scaleDownPolicy.DelaySeconds.ValueInt64()),
	}

	var scaleUpPolicy ScalingPolicyModel
	if diags := data.ScaleUpPolicy.As(ctx, &scaleUpPolicy, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("invalid scale_up_policy")
	}
	scalingOptions.ScaleUpPolicy = &verda.ScalingPolicy{
		DelaySeconds: int(scaleUpPolicy.DelaySeconds.ValueInt64()),
	}

	var queueLoad QueueLoadTriggerModel
	if diags := data.QueueLoad.As(ctx, &queueLoad, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("invalid queue_load")
	}
	scalingOptions.ScalingTriggers = &verda.ScalingTriggers{
		QueueLoad: &verda.QueueLoadTrigger{
			Threshold: queueLoad.Threshold.ValueFloat64(),
		},
	}

	if !data.CPUUtilization.IsNull() && !data.CPUUtilization.IsUnknown() {
		var cpuUtilization UtilizationTriggerModel
		if diags := data.CPUUtilization.As(ctx, &cpuUtilization, basetypes.ObjectAsOptions{}); diags.HasError() {
			return fmt.Errorf("invalid cpu_utilization")
		}
		scalingOptions.ScalingTriggers.CPUUtilization = &verda.UtilizationTrigger{
			Enabled:   cpuUtilization.Enabled.ValueBool(),
			Threshold: int(cpuUtilization.Threshold.ValueInt64()),
		}
	}

	if !data.GPUUtilization.IsNull() && !data.GPUUtilization.IsUnknown() {
		var gpuUtilization UtilizationTriggerModel
		if diags := data.GPUUtilization.As(ctx, &gpuUtilization, basetypes.ObjectAsOptions{}); diags.HasError() {
			return fmt.Errorf("invalid gpu_utilization")
		}
		scalingOptions.ScalingTriggers.GPUUtilization = &verda.UtilizationTrigger{
			Enabled:   gpuUtilization.Enabled.ValueBool(),
			Threshold: int(gpuUtilization.Threshold.ValueInt64()),
		}
	}

	_, err := r.client.ContainerDeployments.UpdateDeploymentScaling(
		ctx,
		data.DeploymentName.ValueString(),
		(*verda.UpdateScalingOptionsRequest)(&scalingOptions),
	)
	return err
}

func (r *ContainerScalingResource) readScaling(ctx context.Context, data *ContainerScalingResourceModel, diagnostics *diag.Diagnostics) {
	scaling, err := r.client.ContainerDeployments.GetDeploymentScaling(ctx, data.DeploymentName.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container scaling, got error: %s", err))
		return
	}

	data.MinReplicaCount = types.Int64Value(int64(scaling.MinReplicaCount))
	data.MaxReplicaCount = types.Int64Value(int64(scaling.MaxReplicaCount))
	data.QueueMessageTTLSeconds = types.Int64Value(int64(scaling.QueueMessageTTLSeconds))
	data.ConcurrentRequestsPerReplica = types.Int64Value(int64(scaling.ConcurrentRequestsPerReplica))
	data.ScaleDownPolicy = scalingPolicyObject(scaling.ScaleDownPolicy, diagnostics)
	data.ScaleUpPolicy = scalingPolicyObject(scaling.ScaleUpPolicy, diagnostics)
	data.QueueLoad = queueLoadObject(scaling.ScalingTriggers, diagnostics)
	data.CPUUtilization = utilizationObject(scaling.ScalingTriggers, true, diagnostics)
	data.GPUUtilization = utilizationObject(scaling.ScalingTriggers, false, diagnostics)
}
