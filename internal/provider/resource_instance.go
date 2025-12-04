package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &InstanceResource{}
var _ resource.ResourceWithImportState = &InstanceResource{}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

type InstanceResource struct {
	client *verda.Client
}

type InstanceResourceModel struct {
	ID              types.String  `tfsdk:"id"`
	InstanceType    types.String  `tfsdk:"instance_type"`
	Image           types.String  `tfsdk:"image"`
	Hostname        types.String  `tfsdk:"hostname"`
	Description     types.String  `tfsdk:"description"`
	PricePerHour    types.Float64 `tfsdk:"price_per_hour"`
	IP              types.String  `tfsdk:"ip"`
	Status          types.String  `tfsdk:"status"`
	CreatedAt       types.String  `tfsdk:"created_at"`
	SSHKeyIDs       types.List    `tfsdk:"ssh_key_ids"`
	Location        types.String  `tfsdk:"location"`
	IsSpot          types.Bool    `tfsdk:"is_spot"`
	OSName          types.String  `tfsdk:"os_name"`
	StartupScriptID types.String  `tfsdk:"startup_script_id"`
	OSVolumeID      types.String  `tfsdk:"os_volume_id"`
	Contract        types.String  `tfsdk:"contract"`
	Pricing         types.String  `tfsdk:"pricing"`
	CPU             types.Object  `tfsdk:"cpu"`
	GPU             types.Object  `tfsdk:"gpu"`
	Memory          types.Object  `tfsdk:"memory"`
	GPUMemory       types.Object  `tfsdk:"gpu_memory"`
	Storage         types.Object  `tfsdk:"storage"`
	JupyterToken    types.String  `tfsdk:"jupyter_token"`
	Volumes         types.List    `tfsdk:"volumes"`
	ExistingVolumes types.List    `tfsdk:"existing_volumes"`
	OSVolume        types.Object  `tfsdk:"os_volume"`
}

type CPUModel struct {
	Description   types.String `tfsdk:"description"`
	NumberOfCores types.Int64  `tfsdk:"number_of_cores"`
}

type GPUModel struct {
	Description  types.String `tfsdk:"description"`
	NumberOfGPUs types.Int64  `tfsdk:"number_of_gpus"`
}

type MemoryModel struct {
	Description     types.String `tfsdk:"description"`
	SizeInGigabytes types.Int64  `tfsdk:"size_in_gigabytes"`
}

type StorageModel struct {
	Description types.String `tfsdk:"description"`
}

type VolumeCreateModel struct {
	Name     types.String `tfsdk:"name"`
	Size     types.Int64  `tfsdk:"size"`
	Type     types.String `tfsdk:"type"`
	Location types.String `tfsdk:"location"`
}

type OSVolumeCreateModel struct {
	Name types.String `tfsdk:"name"`
	Size types.Int64  `tfsdk:"size"`
	Type types.String `tfsdk:"type"`
}

func (r *InstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda compute instance",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Instance identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_type": schema.StringAttribute{
				MarkdownDescription: "Type of the instance (e.g., 'small', 'medium', 'large')",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Image to use for the instance",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname for the instance",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the instance",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"price_per_hour": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "Price per hour for the instance",
			},
			"ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IP address of the instance",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current status of the instance",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
			},
			"ssh_key_ids": schema.ListAttribute{
				MarkdownDescription: "List of SSH key IDs to add to the instance",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Location code for the instance (defaults to FIN-01)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_spot": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a spot instance",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					planmodifier.Bool(&boolDefaultModifier{defaultValue: false}),
				},
			},
			"os_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Operating system name",
			},
			"startup_script_id": schema.StringAttribute{
				MarkdownDescription: "ID of the startup script to run on instance creation",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os_volume_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the OS volume",
			},
			"contract": schema.StringAttribute{
				MarkdownDescription: "Contract type for the instance",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pricing": schema.StringAttribute{
				MarkdownDescription: "Pricing model for the instance",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cpu": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "CPU information",
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Computed: true,
					},
					"number_of_cores": schema.Int64Attribute{
						Computed: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"gpu": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "GPU information",
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Computed: true,
					},
					"number_of_gpus": schema.Int64Attribute{
						Computed: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"memory": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Memory information",
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Computed: true,
					},
					"size_in_gigabytes": schema.Int64Attribute{
						Computed: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"gpu_memory": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "GPU memory information",
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Computed: true,
					},
					"size_in_gigabytes": schema.Int64Attribute{
						Computed: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"storage": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Storage information",
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Computed: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"jupyter_token": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Jupyter token if applicable",
				Sensitive:           true,
			},
			"volumes": schema.ListNestedAttribute{
				MarkdownDescription: "Volumes to create and attach to the instance",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"size": schema.Int64Attribute{
							Required: true,
						},
						"type": schema.StringAttribute{
							Required: true,
						},
						"location": schema.StringAttribute{
							Optional: true,
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"existing_volumes": schema.ListAttribute{
				MarkdownDescription: "IDs of existing volumes to attach to the instance",
				ElementType:         types.StringType,
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"os_volume": schema.SingleNestedAttribute{
				MarkdownDescription: "OS volume configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
					},
					"size": schema.Int64Attribute{
						Required: true,
					},
					"type": schema.StringAttribute{
						Required: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *InstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := verda.CreateInstanceRequest{
		InstanceType: data.InstanceType.ValueString(),
		Image:        data.Image.ValueString(),
		Hostname:     data.Hostname.ValueString(),
		Description:  data.Description.ValueString(),
		LocationCode: data.Location.ValueString(),
		IsSpot:       data.IsSpot.ValueBool(),
	}

	if !data.Contract.IsNull() {
		createReq.Contract = data.Contract.ValueString()
	}

	if !data.Pricing.IsNull() {
		createReq.Pricing = data.Pricing.ValueString()
	}

	if !data.StartupScriptID.IsNull() {
		scriptID := data.StartupScriptID.ValueString()
		createReq.StartupScriptID = &scriptID
	}

	if !data.SSHKeyIDs.IsNull() {
		var sshKeyIDs []string
		resp.Diagnostics.Append(data.SSHKeyIDs.ElementsAs(ctx, &sshKeyIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SSHKeyIDs = sshKeyIDs
	}

	if !data.Volumes.IsNull() {
		var volumes []VolumeCreateModel
		resp.Diagnostics.Append(data.Volumes.ElementsAs(ctx, &volumes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var volumeReqs []verda.VolumeCreateRequest
		for _, vol := range volumes {
			volumeReqs = append(volumeReqs, verda.VolumeCreateRequest{
				Name:         vol.Name.ValueString(),
				Size:         int(vol.Size.ValueInt64()),
				Type:         vol.Type.ValueString(),
				LocationCode: vol.Location.ValueString(),
			})
		}
		createReq.Volumes = volumeReqs
	}

	if !data.ExistingVolumes.IsNull() {
		var existingVolumes []string
		resp.Diagnostics.Append(data.ExistingVolumes.ElementsAs(ctx, &existingVolumes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.ExistingVolumes = existingVolumes
	}

	if !data.OSVolume.IsNull() {
		var osVolume OSVolumeCreateModel
		resp.Diagnostics.Append(data.OSVolume.As(ctx, &osVolume, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		createReq.OSVolume = &verda.OSVolumeCreateRequest{
			Name: osVolume.Name.ValueString(),
			Size: int(osVolume.Size.ValueInt64()),
		}
	}

	instance, err := r.client.Instances.Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create instance, got error: %s", err))
		return
	}

	// Save the instance ID to state immediately to prevent duplicate creation
	// even if subsequent operations fail
	data.ID = types.StringValue(instance.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Now populate the rest of the instance data
	r.flattenInstanceToModel(ctx, instance, &data, &resp.Diagnostics)

	// Update state with full instance details (even if there were non-critical errors)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.client.Instances.GetByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance, got error: %s", err))
		return
	}

	r.flattenInstanceToModel(ctx, instance, &data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data InstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Instances cannot be updated in the Verda API, only deleted and recreated
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Instances cannot be updated for now. Most changes require replacing the resource.",
	)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data InstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Instances.Delete(ctx, data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete instance, got error: %s", err))
		return
	}
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *InstanceResource) flattenInstanceToModel(ctx context.Context, instance *verda.Instance, data *InstanceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(instance.ID)
	data.InstanceType = types.StringValue(instance.InstanceType)
	data.Image = types.StringValue(instance.Image)
	data.Hostname = types.StringValue(instance.Hostname)
	data.Description = types.StringValue(instance.Description)
	data.PricePerHour = types.Float64Value(instance.PricePerHour.Float64())
	data.Status = types.StringValue(instance.Status)
	data.CreatedAt = types.StringValue(instance.CreatedAt.Format("2006-01-02T15:04:05Z"))
	data.Location = types.StringValue(instance.Location)
	data.IsSpot = types.BoolValue(instance.IsSpot)
	data.OSName = types.StringValue(instance.OSName)
	data.Contract = types.StringValue(instance.Contract)
	data.Pricing = types.StringValue(instance.Pricing)

	if instance.IP != nil {
		data.IP = types.StringValue(*instance.IP)
	} else {
		data.IP = types.StringNull()
	}

	if instance.OSVolumeID != nil {
		data.OSVolumeID = types.StringValue(*instance.OSVolumeID)
	} else {
		data.OSVolumeID = types.StringNull()
	}

	if instance.StartupScriptID != nil {
		data.StartupScriptID = types.StringValue(*instance.StartupScriptID)
	} else {
		data.StartupScriptID = types.StringNull()
	}

	if instance.JupyterToken != nil {
		data.JupyterToken = types.StringValue(*instance.JupyterToken)
	} else {
		data.JupyterToken = types.StringNull()
	}

	sshKeyList, diags := types.ListValueFrom(ctx, types.StringType, instance.SSHKeyIDs)
	diagnostics.Append(diags...)
	data.SSHKeyIDs = sshKeyList

	cpuObj, cpuDiags := types.ObjectValue(
		map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		},
		map[string]attr.Value{
			"description":     types.StringValue(instance.CPU.Description),
			"number_of_cores": types.Int64Value(int64(instance.CPU.NumberOfCores)),
		},
	)
	diagnostics.Append(cpuDiags...)
	data.CPU = cpuObj

	gpuObj, gpuDiags := types.ObjectValue(
		map[string]attr.Type{
			"description":    types.StringType,
			"number_of_gpus": types.Int64Type,
		},
		map[string]attr.Value{
			"description":    types.StringValue(instance.GPU.Description),
			"number_of_gpus": types.Int64Value(int64(instance.GPU.NumberOfGPUs)),
		},
	)
	diagnostics.Append(gpuDiags...)
	data.GPU = gpuObj

	gpuMemoryObj, gpuMemDiags := types.ObjectValue(
		map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		},
		map[string]attr.Value{
			"description":       types.StringValue(instance.GPUMemory.Description),
			"size_in_gigabytes": types.Int64Value(int64(instance.GPUMemory.SizeInGigabytes)),
		},
	)
	diagnostics.Append(gpuMemDiags...)
	data.GPUMemory = gpuMemoryObj

	memoryObj, memDiags := types.ObjectValue(
		map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		},
		map[string]attr.Value{
			"description":       types.StringValue(instance.Memory.Description),
			"size_in_gigabytes": types.Int64Value(int64(instance.Memory.SizeInGigabytes)),
		},
	)
	diagnostics.Append(memDiags...)
	data.Memory = memoryObj

	storageObj, storDiags := types.ObjectValue(
		map[string]attr.Type{
			"description": types.StringType,
		},
		map[string]attr.Value{
			"description": types.StringValue(instance.Storage.Description),
		},
	)
	diagnostics.Append(storDiags...)
	data.Storage = storageObj
}

// Custom plan modifier for bool default value
type boolDefaultModifier struct {
	defaultValue bool
}

func (m *boolDefaultModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("Sets the default value to %t if not configured", m.defaultValue)
}

func (m *boolDefaultModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Sets the default value to `%t` if not configured", m.defaultValue)
}

func (m *boolDefaultModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		resp.PlanValue = types.BoolValue(m.defaultValue)
	}
}
