package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

// Container deployment status data source

type ContainerDeploymentStatusDataSource struct {
	client *verda.Client
}

type ContainerDeploymentStatusDataSourceModel struct {
	DeploymentName    types.String `tfsdk:"deployment_name"`
	Status            types.String `tfsdk:"status"`
	DesiredReplicas   types.Int64  `tfsdk:"desired_replicas"`
	CurrentReplicas   types.Int64  `tfsdk:"current_replicas"`
	AvailableReplicas types.Int64  `tfsdk:"available_replicas"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func NewContainerDeploymentStatusDataSource() datasource.DataSource {
	return &ContainerDeploymentStatusDataSource{}
}

func (d *ContainerDeploymentStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_deployment_status"
}

func (d *ContainerDeploymentStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches deployment status for a container deployment.",
		Attributes: map[string]schema.Attribute{
			"deployment_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Deployment name.",
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"desired_replicas": schema.Int64Attribute{
				Computed: true,
			},
			"current_replicas": schema.Int64Attribute{
				Computed: true,
			},
			"available_replicas": schema.Int64Attribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *ContainerDeploymentStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContainerDeploymentStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerDeploymentStatusDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, err := d.client.ContainerDeployments.GetDeploymentStatus(ctx, data.DeploymentName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container deployment status, got error: %s", err))
		return
	}

	data.Status = types.StringValue(status.Status)
	data.DesiredReplicas = types.Int64Value(int64(status.DesiredReplicas))
	data.CurrentReplicas = types.Int64Value(int64(status.CurrentReplicas))
	data.AvailableReplicas = types.Int64Value(int64(status.AvailableReplicas))
	data.UpdatedAt = types.StringValue(status.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Container deployment replicas data source

type ContainerDeploymentReplicasDataSource struct {
	client *verda.Client
}

type ContainerDeploymentReplicasDataSourceModel struct {
	DeploymentName types.String `tfsdk:"deployment_name"`
	Replicas       types.List   `tfsdk:"replicas"`
}

func NewContainerDeploymentReplicasDataSource() datasource.DataSource {
	return &ContainerDeploymentReplicasDataSource{}
}

func (d *ContainerDeploymentReplicasDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_deployment_replicas"
}

func (d *ContainerDeploymentReplicasDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches replicas for a container deployment.",
		Attributes: map[string]schema.Attribute{
			"deployment_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Deployment name.",
			},
			"replicas": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Replica details.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"status": schema.StringAttribute{
							Computed: true,
						},
						"node": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ContainerDeploymentReplicasDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContainerDeploymentReplicasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerDeploymentReplicasDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	replicas, err := d.client.ContainerDeployments.GetDeploymentReplicas(ctx, data.DeploymentName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container deployment replicas, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"name":   types.StringType,
		"status": types.StringType,
		"node":   types.StringType,
	}

	var items []map[string]attr.Value
	for _, replica := range replicas.Replicas {
		items = append(items, map[string]attr.Value{
			"name":   types.StringValue(replica.Name),
			"status": types.StringValue(replica.Status),
			"node":   types.StringValue(replica.Node),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Replicas = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Container deployment scaling data source

type ContainerDeploymentScalingDataSource struct {
	client *verda.Client
}

type ContainerDeploymentScalingDataSourceModel struct {
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

func NewContainerDeploymentScalingDataSource() datasource.DataSource {
	return &ContainerDeploymentScalingDataSource{}
}

func (d *ContainerDeploymentScalingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_deployment_scaling"
}

func (d *ContainerDeploymentScalingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	utilizationAttrs := map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{Computed: true},
		"threshold": schema.Int64Attribute{
			Computed: true,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches scaling configuration for a container deployment.",
		Attributes: map[string]schema.Attribute{
			"deployment_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Deployment name.",
			},
			"min_replica_count": schema.Int64Attribute{Computed: true},
			"max_replica_count": schema.Int64Attribute{Computed: true},
			"queue_message_ttl_seconds": schema.Int64Attribute{
				Computed: true,
			},
			"concurrent_requests_per_replica": schema.Int64Attribute{
				Computed: true,
			},
			"scale_down_policy": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"delay_seconds": schema.Int64Attribute{Computed: true},
				},
			},
			"scale_up_policy": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"delay_seconds": schema.Int64Attribute{Computed: true},
				},
			},
			"queue_load": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"threshold": schema.Float64Attribute{Computed: true},
				},
			},
			"cpu_utilization": schema.SingleNestedAttribute{
				Computed:   true,
				Attributes: utilizationAttrs,
			},
			"gpu_utilization": schema.SingleNestedAttribute{
				Computed:   true,
				Attributes: utilizationAttrs,
			},
		},
	}
}

func (d *ContainerDeploymentScalingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContainerDeploymentScalingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerDeploymentScalingDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scaling, err := d.client.ContainerDeployments.GetDeploymentScaling(ctx, data.DeploymentName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container deployment scaling, got error: %s", err))
		return
	}

	data.MinReplicaCount = types.Int64Value(int64(scaling.MinReplicaCount))
	data.MaxReplicaCount = types.Int64Value(int64(scaling.MaxReplicaCount))
	data.QueueMessageTTLSeconds = types.Int64Value(int64(scaling.QueueMessageTTLSeconds))
	data.ConcurrentRequestsPerReplica = types.Int64Value(int64(scaling.ConcurrentRequestsPerReplica))
	data.ScaleDownPolicy = scalingPolicyObject(scaling.ScaleDownPolicy, &resp.Diagnostics)
	data.ScaleUpPolicy = scalingPolicyObject(scaling.ScaleUpPolicy, &resp.Diagnostics)
	data.QueueLoad = queueLoadObject(scaling.ScalingTriggers, &resp.Diagnostics)
	data.CPUUtilization = utilizationObject(scaling.ScalingTriggers, true, &resp.Diagnostics)
	data.GPUUtilization = utilizationObject(scaling.ScalingTriggers, false, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Job deployment status data source

type JobDeploymentStatusDataSource struct {
	client *verda.Client
}

type JobDeploymentStatusDataSourceModel struct {
	JobName       types.String `tfsdk:"job_name"`
	Status        types.String `tfsdk:"status"`
	ActiveJobs    types.Int64  `tfsdk:"active_jobs"`
	SucceededJobs types.Int64  `tfsdk:"succeeded_jobs"`
	FailedJobs    types.Int64  `tfsdk:"failed_jobs"`
}

func NewJobDeploymentStatusDataSource() datasource.DataSource {
	return &JobDeploymentStatusDataSource{}
}

func (d *JobDeploymentStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_deployment_status"
}

func (d *JobDeploymentStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches status for a job deployment.",
		Attributes: map[string]schema.Attribute{
			"job_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Job deployment name.",
			},
			"status": schema.StringAttribute{Computed: true},
			"active_jobs": schema.Int64Attribute{
				Computed: true,
			},
			"succeeded_jobs": schema.Int64Attribute{
				Computed: true,
			},
			"failed_jobs": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (d *JobDeploymentStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *JobDeploymentStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data JobDeploymentStatusDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, err := d.client.ServerlessJobs.GetJobDeploymentStatus(ctx, data.JobName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read job deployment status, got error: %s", err))
		return
	}

	data.Status = types.StringValue(status.Status)
	data.ActiveJobs = types.Int64Value(int64(status.ActiveJobs))
	data.SucceededJobs = types.Int64Value(int64(status.SucceededJobs))
	data.FailedJobs = types.Int64Value(int64(status.FailedJobs))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Job deployment scaling data source

type JobDeploymentScalingDataSource struct {
	client *verda.Client
}

type JobDeploymentScalingDataSourceModel struct {
	JobName                      types.String `tfsdk:"job_name"`
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

func NewJobDeploymentScalingDataSource() datasource.DataSource {
	return &JobDeploymentScalingDataSource{}
}

func (d *JobDeploymentScalingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_deployment_scaling"
}

func (d *JobDeploymentScalingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	utilizationAttrs := map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{Computed: true},
		"threshold": schema.Int64Attribute{
			Computed: true,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches scaling configuration for a job deployment.",
		Attributes: map[string]schema.Attribute{
			"job_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Job deployment name.",
			},
			"min_replica_count": schema.Int64Attribute{Computed: true},
			"max_replica_count": schema.Int64Attribute{Computed: true},
			"queue_message_ttl_seconds": schema.Int64Attribute{
				Computed: true,
			},
			"concurrent_requests_per_replica": schema.Int64Attribute{
				Computed: true,
			},
			"scale_down_policy": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"delay_seconds": schema.Int64Attribute{Computed: true},
				},
			},
			"scale_up_policy": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"delay_seconds": schema.Int64Attribute{Computed: true},
				},
			},
			"queue_load": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"threshold": schema.Float64Attribute{Computed: true},
				},
			},
			"cpu_utilization": schema.SingleNestedAttribute{
				Computed:   true,
				Attributes: utilizationAttrs,
			},
			"gpu_utilization": schema.SingleNestedAttribute{
				Computed:   true,
				Attributes: utilizationAttrs,
			},
		},
	}
}

func (d *JobDeploymentScalingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *JobDeploymentScalingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data JobDeploymentScalingDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scaling, err := d.client.ServerlessJobs.GetJobDeploymentScaling(ctx, data.JobName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read job deployment scaling, got error: %s", err))
		return
	}

	data.MinReplicaCount = types.Int64Value(int64(scaling.MinReplicaCount))
	data.MaxReplicaCount = types.Int64Value(int64(scaling.MaxReplicaCount))
	data.QueueMessageTTLSeconds = types.Int64Value(int64(scaling.QueueMessageTTLSeconds))
	data.ConcurrentRequestsPerReplica = types.Int64Value(int64(scaling.ConcurrentRequestsPerReplica))
	data.ScaleDownPolicy = scalingPolicyObject(scaling.ScaleDownPolicy, &resp.Diagnostics)
	data.ScaleUpPolicy = scalingPolicyObject(scaling.ScaleUpPolicy, &resp.Diagnostics)
	data.QueueLoad = queueLoadObject(scaling.ScalingTriggers, &resp.Diagnostics)
	data.CPUUtilization = utilizationObject(scaling.ScalingTriggers, true, &resp.Diagnostics)
	data.GPUUtilization = utilizationObject(scaling.ScalingTriggers, false, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Volume trash data source

type VolumeTrashDataSource struct {
	client *verda.Client
}

type VolumeTrashDataSourceModel struct {
	Volumes types.List `tfsdk:"volumes"`
}

type volumeTrashResponse struct {
	ID             string   `json:"id"`
	InstanceID     string   `json:"instance_id"`
	Instances      []string `json:"instances"`
	Name           string   `json:"name"`
	CreatedAt      string   `json:"created_at"`
	Status         string   `json:"status"`
	Size           float64  `json:"size"`
	IsOSVolume     bool     `json:"is_os_volume"`
	Target         string   `json:"target"`
	Type           string   `json:"type"`
	Location       string   `json:"location"`
	SSHKeyIDs      []string `json:"ssh_key_ids"`
	Contract       string   `json:"contract"`
	BaseHourlyCost float64  `json:"base_hourly_cost"`
	MonthlyPrice   float64  `json:"monthly_price"`
	Currency       string   `json:"currency"`
	DeletedAt      string   `json:"deleted_at"`
}

func NewVolumeTrashDataSource() datasource.DataSource {
	return &VolumeTrashDataSource{}
}

func (d *VolumeTrashDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volumes_trash"
}

func (d *VolumeTrashDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches volumes currently in trash.",
		Attributes: map[string]schema.Attribute{
			"volumes": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Volumes in trash.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"instance_id": schema.StringAttribute{Computed: true},
						"instances": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"name":       schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{Computed: true},
						"status":     schema.StringAttribute{Computed: true},
						"size":       schema.Float64Attribute{Computed: true},
						"is_os_volume": schema.BoolAttribute{
							Computed: true,
						},
						"target":   schema.StringAttribute{Computed: true},
						"type":     schema.StringAttribute{Computed: true},
						"location": schema.StringAttribute{Computed: true},
						"ssh_key_ids": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"contract": schema.StringAttribute{Computed: true},
						"base_hourly_cost": schema.Float64Attribute{
							Computed: true,
						},
						"monthly_price": schema.Float64Attribute{
							Computed: true,
						},
						"currency": schema.StringAttribute{Computed: true},
						"deleted_at": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *VolumeTrashDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VolumeTrashDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VolumeTrashDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var response []volumeTrashResponse
	if err := doVerdaRequest(ctx, d.client, "GET", "/volumes/trash", nil, &response); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volumes in trash, got error: %s", err))
		return
	}

	attrTypes := map[string]attr.Type{
		"id":               types.StringType,
		"instance_id":      types.StringType,
		"instances":        types.ListType{ElemType: types.StringType},
		"name":             types.StringType,
		"created_at":       types.StringType,
		"status":           types.StringType,
		"size":             types.Float64Type,
		"is_os_volume":     types.BoolType,
		"target":           types.StringType,
		"type":             types.StringType,
		"location":         types.StringType,
		"ssh_key_ids":      types.ListType{ElemType: types.StringType},
		"contract":         types.StringType,
		"base_hourly_cost": types.Float64Type,
		"monthly_price":    types.Float64Type,
		"currency":         types.StringType,
		"deleted_at":       types.StringType,
	}

	var items []map[string]attr.Value
	for _, volume := range response {
		instanceList, diags := stringListValue(ctx, volume.Instances)
		resp.Diagnostics.Append(diags...)
		sshKeyList, diags := stringListValue(ctx, volume.SSHKeyIDs)
		resp.Diagnostics.Append(diags...)

		items = append(items, map[string]attr.Value{
			"id":               types.StringValue(volume.ID),
			"instance_id":      stringValueOrNull(volume.InstanceID),
			"instances":        instanceList,
			"name":             types.StringValue(volume.Name),
			"created_at":       types.StringValue(volume.CreatedAt),
			"status":           types.StringValue(volume.Status),
			"size":             types.Float64Value(volume.Size),
			"is_os_volume":     types.BoolValue(volume.IsOSVolume),
			"target":           stringValueOrNull(volume.Target),
			"type":             types.StringValue(volume.Type),
			"location":         stringValueOrNull(volume.Location),
			"ssh_key_ids":      sshKeyList,
			"contract":         stringValueOrNull(volume.Contract),
			"base_hourly_cost": types.Float64Value(volume.BaseHourlyCost),
			"monthly_price":    types.Float64Value(volume.MonthlyPrice),
			"currency":         types.StringValue(volume.Currency),
			"deleted_at":       stringValueOrNull(volume.DeletedAt),
		})
	}

	listValue, diags := objectListValue(attrTypes, items)
	resp.Diagnostics.Append(diags...)
	data.Volumes = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Deprecated instance endpoints data sources

type InstanceTypesDeprecatedDataSource struct {
	client *verda.Client
}

type InstanceTypesDeprecatedDataSourceModel struct {
	RawJSON types.String `tfsdk:"raw_json"`
}

func NewInstanceTypesDeprecatedDataSource() datasource.DataSource {
	return &InstanceTypesDeprecatedDataSource{}
}

func (d *InstanceTypesDeprecatedDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_types_deprecated"
}

func (d *InstanceTypesDeprecatedDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches instance types using deprecated endpoint `/v1/instances/types`.",
		Attributes: map[string]schema.Attribute{
			"raw_json": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Raw JSON response from the deprecated endpoint.",
			},
		},
	}
}

func (d *InstanceTypesDeprecatedDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceTypesDeprecatedDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceTypesDeprecatedDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var raw json.RawMessage
	if err := doVerdaRequest(ctx, d.client, "GET", "/instances/types", nil, &raw); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deprecated instance types, got error: %s", err))
		return
	}

	data.RawJSON = types.StringValue(string(raw))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type InstanceAvailabilityDeprecatedDataSource struct {
	client *verda.Client
}

type InstanceAvailabilityDeprecatedDataSourceModel struct {
	InstanceType types.String `tfsdk:"instance_type"`
	Available    types.Bool   `tfsdk:"available"`
	RawJSON      types.String `tfsdk:"raw_json"`
}

func NewInstanceAvailabilityDeprecatedDataSource() datasource.DataSource {
	return &InstanceAvailabilityDeprecatedDataSource{}
}

func (d *InstanceAvailabilityDeprecatedDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_availability_deprecated"
}

func (d *InstanceAvailabilityDeprecatedDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks availability using deprecated endpoint `/v1/instances/availability/{instanceType}`.",
		Attributes: map[string]schema.Attribute{
			"instance_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Instance type to check.",
			},
			"available": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Availability result if the response is boolean.",
			},
			"raw_json": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Raw JSON response.",
			},
		},
	}
}

func (d *InstanceAvailabilityDeprecatedDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceAvailabilityDeprecatedDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceAvailabilityDeprecatedDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := fmt.Sprintf("/instances/availability/%s", url.PathEscape(data.InstanceType.ValueString()))

	var raw json.RawMessage
	if err := doVerdaRequest(ctx, d.client, "GET", path, nil, &raw); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deprecated instance availability, got error: %s", err))
		return
	}

	data.RawJSON = types.StringValue(string(raw))
	var available bool
	if err := json.Unmarshal(raw, &available); err == nil {
		data.Available = types.BoolValue(available)
	} else {
		data.Available = types.BoolNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func scalingPolicyObject(policy *verda.ScalingPolicy, diagnostics *diag.Diagnostics) types.Object {
	if policy == nil {
		return types.ObjectNull(map[string]attr.Type{
			"delay_seconds": types.Int64Type,
		})
	}

	obj, diags := types.ObjectValue(
		map[string]attr.Type{
			"delay_seconds": types.Int64Type,
		},
		map[string]attr.Value{
			"delay_seconds": types.Int64Value(int64(policy.DelaySeconds)),
		},
	)
	diagnostics.Append(diags...)
	return obj
}

func queueLoadObject(triggers *verda.ScalingTriggers, diagnostics *diag.Diagnostics) types.Object {
	if triggers == nil || triggers.QueueLoad == nil {
		return types.ObjectNull(map[string]attr.Type{
			"threshold": types.Float64Type,
		})
	}

	obj, diags := types.ObjectValue(
		map[string]attr.Type{
			"threshold": types.Float64Type,
		},
		map[string]attr.Value{
			"threshold": types.Float64Value(triggers.QueueLoad.Threshold),
		},
	)
	diagnostics.Append(diags...)
	return obj
}

func utilizationObject(triggers *verda.ScalingTriggers, useCPU bool, diagnostics *diag.Diagnostics) types.Object {
	target := (*verda.UtilizationTrigger)(nil)
	if triggers != nil {
		if useCPU {
			target = triggers.CPUUtilization
		} else {
			target = triggers.GPUUtilization
		}
	}

	if target == nil {
		return types.ObjectNull(map[string]attr.Type{
			"enabled":   types.BoolType,
			"threshold": types.Int64Type,
		})
	}

	obj, diags := types.ObjectValue(
		map[string]attr.Type{
			"enabled":   types.BoolType,
			"threshold": types.Int64Type,
		},
		map[string]attr.Value{
			"enabled":   types.BoolValue(target.Enabled),
			"threshold": types.Int64Value(int64(target.Threshold)),
		},
	)
	diagnostics.Append(diags...)
	return obj
}
