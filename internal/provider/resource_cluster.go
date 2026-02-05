package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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

const clusterActionDiscontinue = "discontinue"

var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

type ClusterResource struct {
	client *verda.Client
}

type ClusterResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ClusterType         types.String `tfsdk:"cluster_type"`
	Image               types.String `tfsdk:"image"`
	Hostname            types.String `tfsdk:"hostname"`
	Description         types.String `tfsdk:"description"`
	SharedVolume        types.Object `tfsdk:"shared_volume"`
	ExistingVolumes     types.List   `tfsdk:"existing_volumes"`
	Location            types.String `tfsdk:"location"`
	Contract            types.String `tfsdk:"contract"`
	SSHKeyIDs           types.List   `tfsdk:"ssh_key_ids"`
	StartupScriptID     types.String `tfsdk:"startup_script_id"`
	AutoRentalExtension types.Bool   `tfsdk:"auto_rental_extension"`
	TurnToPayAsYouGo    types.Bool   `tfsdk:"turn_to_pay_as_you_go"`

	IP             types.String  `tfsdk:"ip"`
	Status         types.String  `tfsdk:"status"`
	CreatedAt      types.String  `tfsdk:"created_at"`
	PricePerHour   types.Float64 `tfsdk:"price_per_hour"`
	OSName         types.String  `tfsdk:"os_name"`
	CPU            types.Object  `tfsdk:"cpu"`
	GPU            types.Object  `tfsdk:"gpu"`
	Memory         types.Object  `tfsdk:"memory"`
	GPUMemory      types.Object  `tfsdk:"gpu_memory"`
	Storage        types.Object  `tfsdk:"storage"`
	SharedVolumes  types.List    `tfsdk:"shared_volumes"`
	LongTermPeriod types.String  `tfsdk:"long_term_period"`
	WorkerNodes    types.List    `tfsdk:"worker_nodes"`
}

type SharedVolumeModel struct {
	Name types.String `tfsdk:"name"`
	Size types.Int64  `tfsdk:"size"`
}

type ExistingSharedVolumeModel struct {
	ID types.String `tfsdk:"id"`
}

type deployClusterRequest struct {
	ClusterType         string                        `json:"cluster_type"`
	Image               string                        `json:"image"`
	Hostname            string                        `json:"hostname"`
	Description         string                        `json:"description"`
	SharedVolume        sharedVolumeRequest           `json:"shared_volume"`
	ExistingVolumes     []existingSharedVolumeRequest `json:"existing_volumes,omitempty"`
	LocationCode        string                        `json:"location_code,omitempty"`
	Contract            string                        `json:"contract,omitempty"`
	SSHKeyIDs           []string                      `json:"ssh_key_ids,omitempty"`
	StartupScriptID     *string                       `json:"startup_script_id,omitempty"`
	AutoRentalExtension *bool                         `json:"auto_rental_extension,omitempty"`
	TurnToPayAsYouGo    *bool                         `json:"turn_to_pay_as_you_go,omitempty"`
}

type sharedVolumeRequest struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type existingSharedVolumeRequest struct {
	ID string `json:"id"`
}

type deployClusterResponse struct {
	ID string `json:"id"`
}

type clusterActionRequest struct {
	Actions []clusterAction `json:"actions"`
}

type clusterAction struct {
	Action string `json:"action"`
	ID     string `json:"id"`
}

type clusterResponse struct {
	ID                  string                `json:"id"`
	ClusterType         string                `json:"cluster_type"`
	Image               string                `json:"image"`
	PricePerHour        verda.FlexibleFloat   `json:"price_per_hour"`
	Hostname            string                `json:"hostname"`
	Description         string                `json:"description"`
	IP                  *string               `json:"ip"`
	Status              string                `json:"status"`
	CreatedAt           string                `json:"created_at"`
	SSHKeyIDs           []string              `json:"ssh_key_ids"`
	CPU                 verda.InstanceCPU     `json:"cpu"`
	GPU                 verda.InstanceGPU     `json:"gpu"`
	Memory              verda.InstanceMemory  `json:"memory"`
	Storage             verda.InstanceStorage `json:"storage"`
	GPUMemory           verda.InstanceMemory  `json:"gpu_memory"`
	Location            string                `json:"location"`
	OSName              string                `json:"os_name"`
	StartupScriptID     *string               `json:"startup_script_id"`
	Contract            string                `json:"contract"`
	AutoRentalExtension *bool                 `json:"auto_rental_extension"`
	TurnToPayAsYouGo    *bool                 `json:"turn_to_pay_as_you_go"`
	LongTermPeriod      *string               `json:"long_term_period"`
	SharedVolumes       []clusterSharedVolume `json:"shared_volumes"`
	WorkerNodes         []clusterWorkerNode   `json:"worker_nodes"`
}

type clusterSharedVolume struct {
	ID              string `json:"id"`
	MountPoint      string `json:"mount_point"`
	Name            string `json:"name"`
	SizeInGigabytes int    `json:"size_in_gigabytes"`
}

func (s *clusterSharedVolume) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return err
		}
		s.ID = id
		return nil
	}

	type alias clusterSharedVolume
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = clusterSharedVolume(decoded)
	return nil
}

type clusterWorkerNode struct {
	ID        string `json:"id"`
	Hostname  string `json:"hostname"`
	Status    string `json:"status"`
	PrivateIP string `json:"private_ip"`
	PublicIP  string `json:"public_ip"`
}

func (w *clusterWorkerNode) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return err
		}
		w.ID = id
		return nil
	}

	type alias clusterWorkerNode
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*w = clusterWorkerNode(decoded)
	return nil
}

func (r *ClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *ClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda cluster deployment.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_type": schema.StringAttribute{
				MarkdownDescription: "Cluster type (e.g., 16H200).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Cluster image.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Cluster hostname.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Cluster description.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"shared_volume": schema.SingleNestedAttribute{
				MarkdownDescription: "Shared cluster volume configuration.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Shared volume name.",
						Required:            true,
					},
					"size": schema.Int64Attribute{
						MarkdownDescription: "Shared volume size in GB.",
						Required:            true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"existing_volumes": schema.ListNestedAttribute{
				MarkdownDescription: "Existing shared volumes to attach.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Existing shared volume ID.",
							Required:            true,
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Location code for the cluster (defaults to FIN-01).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract": schema.StringAttribute{
				MarkdownDescription: "Contract type for the cluster.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_key_ids": schema.ListAttribute{
				MarkdownDescription: "SSH key IDs to attach.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"startup_script_id": schema.StringAttribute{
				MarkdownDescription: "Startup script ID to run on cluster initialization.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auto_rental_extension": schema.BoolAttribute{
				MarkdownDescription: "Enable automatic rental extension for long-term clusters.",
				Optional:            true,
				Computed:            true,
			},
			"turn_to_pay_as_you_go": schema.BoolAttribute{
				MarkdownDescription: "Convert to pay-as-you-go after the long-term period ends.",
				Optional:            true,
				Computed:            true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "Cluster jump host IP address.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Cluster status.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp.",
				Computed:            true,
			},
			"price_per_hour": schema.Float64Attribute{
				MarkdownDescription: "Cluster price per hour.",
				Computed:            true,
			},
			"os_name": schema.StringAttribute{
				MarkdownDescription: "Operating system name.",
				Computed:            true,
			},
			"cpu": schema.SingleNestedAttribute{
				MarkdownDescription: "CPU information.",
				Computed:            true,
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
				MarkdownDescription: "GPU information.",
				Computed:            true,
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
				MarkdownDescription: "Memory information.",
				Computed:            true,
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
				MarkdownDescription: "GPU memory information.",
				Computed:            true,
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
				MarkdownDescription: "Storage information.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Computed: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"shared_volumes": schema.ListNestedAttribute{
				MarkdownDescription: "Shared volumes attached to the cluster.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"mount_point": schema.StringAttribute{
							Computed: true,
						},
						"size_in_gigabytes": schema.Int64Attribute{
							Computed: true,
						},
					},
				},
			},
			"long_term_period": schema.StringAttribute{
				MarkdownDescription: "Long-term rental period description.",
				Computed:            true,
			},
			"worker_nodes": schema.ListNestedAttribute{
				MarkdownDescription: "Worker nodes in the cluster.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"hostname": schema.StringAttribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
						"private_ip": schema.StringAttribute{
							Computed: true,
						},
						"public_ip": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (r *ClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sharedVolume SharedVolumeModel
	resp.Diagnostics.Append(data.SharedVolume.As(ctx, &sharedVolume, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterReq := deployClusterRequest{
		ClusterType: data.ClusterType.ValueString(),
		Image:       data.Image.ValueString(),
		Hostname:    data.Hostname.ValueString(),
		Description: data.Description.ValueString(),
		SharedVolume: sharedVolumeRequest{
			Name: sharedVolume.Name.ValueString(),
			Size: int(sharedVolume.Size.ValueInt64()),
		},
	}

	if !data.Location.IsNull() {
		clusterReq.LocationCode = data.Location.ValueString()
	}

	if !data.Contract.IsNull() {
		clusterReq.Contract = data.Contract.ValueString()
	}

	if !data.StartupScriptID.IsNull() {
		startupScriptID := data.StartupScriptID.ValueString()
		clusterReq.StartupScriptID = &startupScriptID
	}

	if !data.AutoRentalExtension.IsNull() && !data.AutoRentalExtension.IsUnknown() {
		value := data.AutoRentalExtension.ValueBool()
		clusterReq.AutoRentalExtension = &value
	}

	if !data.TurnToPayAsYouGo.IsNull() && !data.TurnToPayAsYouGo.IsUnknown() {
		value := data.TurnToPayAsYouGo.ValueBool()
		clusterReq.TurnToPayAsYouGo = &value
	}

	if !data.SSHKeyIDs.IsNull() {
		var sshKeyIDs []string
		resp.Diagnostics.Append(data.SSHKeyIDs.ElementsAs(ctx, &sshKeyIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		clusterReq.SSHKeyIDs = sshKeyIDs
	}

	if !data.ExistingVolumes.IsNull() {
		var existingVolumes []ExistingSharedVolumeModel
		resp.Diagnostics.Append(data.ExistingVolumes.ElementsAs(ctx, &existingVolumes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var requestVolumes []existingSharedVolumeRequest
		for _, volume := range existingVolumes {
			requestVolumes = append(requestVolumes, existingSharedVolumeRequest{
				ID: volume.ID.ValueString(),
			})
		}
		clusterReq.ExistingVolumes = requestVolumes
	}

	var createResp deployClusterResponse
	if err := doVerdaRequest(ctx, r.client, http.MethodPost, "/clusters", clusterReq, &createResp); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createResp.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.getClusterByID(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster after creation, got error: %s", err))
		return
	}

	r.flattenClusterResponse(ctx, cluster, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	priorSharedVolume := data.SharedVolume
	priorExistingVolumes := data.ExistingVolumes
	priorAutoRental := data.AutoRentalExtension
	priorTurnToPayAsYouGo := data.TurnToPayAsYouGo
	priorStartupScript := data.StartupScriptID

	cluster, err := r.getClusterByID(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster, got error: %s", err))
		return
	}

	r.flattenClusterResponse(ctx, cluster, &data, &resp.Diagnostics)

	if data.SharedVolume.IsNull() || data.SharedVolume.IsUnknown() {
		data.SharedVolume = priorSharedVolume
	}
	if data.ExistingVolumes.IsNull() || data.ExistingVolumes.IsUnknown() {
		data.ExistingVolumes = priorExistingVolumes
	}
	if data.AutoRentalExtension.IsNull() || data.AutoRentalExtension.IsUnknown() {
		data.AutoRentalExtension = priorAutoRental
	}
	if data.TurnToPayAsYouGo.IsNull() || data.TurnToPayAsYouGo.IsUnknown() {
		data.TurnToPayAsYouGo = priorTurnToPayAsYouGo
	}
	if data.StartupScriptID.IsNull() || data.StartupScriptID.IsUnknown() {
		data.StartupScriptID = priorStartupScript
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Clusters cannot be updated. Please delete and recreate the resource.",
	)
}

func (r *ClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionReq := clusterActionRequest{
		Actions: []clusterAction{
			{
				Action: clusterActionDiscontinue,
				ID:     data.ID.ValueString(),
			},
		},
	}

	if err := doVerdaRequest(ctx, r.client, http.MethodPut, "/clusters", actionReq, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to discontinue cluster, got error: %s", err))
		return
	}
}

func (r *ClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ClusterResource) getClusterByID(ctx context.Context, id string) (*clusterResponse, error) {
	var response clusterResponse
	path := fmt.Sprintf("/clusters/%s", url.PathEscape(id))

	if err := doVerdaRequest(ctx, r.client, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (r *ClusterResource) flattenClusterResponse(ctx context.Context, cluster *clusterResponse, data *ClusterResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(cluster.ID)
	data.ClusterType = types.StringValue(cluster.ClusterType)
	data.Image = types.StringValue(cluster.Image)
	data.Hostname = types.StringValue(cluster.Hostname)
	data.Description = types.StringValue(cluster.Description)
	data.Status = types.StringValue(cluster.Status)
	data.CreatedAt = types.StringValue(cluster.CreatedAt)
	data.PricePerHour = types.Float64Value(cluster.PricePerHour.Float64())
	data.OSName = types.StringValue(cluster.OSName)
	data.Location = types.StringValue(cluster.Location)
	data.Contract = types.StringValue(cluster.Contract)

	if cluster.IP != nil && *cluster.IP != "" {
		data.IP = types.StringValue(*cluster.IP)
	} else {
		data.IP = types.StringNull()
	}

	if cluster.StartupScriptID != nil && *cluster.StartupScriptID != "" {
		data.StartupScriptID = types.StringValue(*cluster.StartupScriptID)
	}

	if cluster.AutoRentalExtension != nil {
		data.AutoRentalExtension = types.BoolValue(*cluster.AutoRentalExtension)
	}

	if cluster.TurnToPayAsYouGo != nil {
		data.TurnToPayAsYouGo = types.BoolValue(*cluster.TurnToPayAsYouGo)
	}

	if cluster.LongTermPeriod != nil && *cluster.LongTermPeriod != "" {
		data.LongTermPeriod = types.StringValue(*cluster.LongTermPeriod)
	} else {
		data.LongTermPeriod = types.StringNull()
	}

	if cluster.SSHKeyIDs != nil {
		sshKeyList, diags := types.ListValueFrom(ctx, types.StringType, cluster.SSHKeyIDs)
		diagnostics.Append(diags...)
		data.SSHKeyIDs = sshKeyList
	}

	cpuObj, cpuDiags := types.ObjectValue(
		map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		},
		map[string]attr.Value{
			"description":     types.StringValue(cluster.CPU.Description),
			"number_of_cores": types.Int64Value(int64(cluster.CPU.NumberOfCores)),
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
			"description":    types.StringValue(cluster.GPU.Description),
			"number_of_gpus": types.Int64Value(int64(cluster.GPU.NumberOfGPUs)),
		},
	)
	diagnostics.Append(gpuDiags...)
	data.GPU = gpuObj

	memoryObj, memDiags := types.ObjectValue(
		map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		},
		map[string]attr.Value{
			"description":       types.StringValue(cluster.Memory.Description),
			"size_in_gigabytes": types.Int64Value(int64(cluster.Memory.SizeInGigabytes)),
		},
	)
	diagnostics.Append(memDiags...)
	data.Memory = memoryObj

	gpuMemoryObj, gpuMemDiags := types.ObjectValue(
		map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		},
		map[string]attr.Value{
			"description":       types.StringValue(cluster.GPUMemory.Description),
			"size_in_gigabytes": types.Int64Value(int64(cluster.GPUMemory.SizeInGigabytes)),
		},
	)
	diagnostics.Append(gpuMemDiags...)
	data.GPUMemory = gpuMemoryObj

	storageObj, storageDiags := types.ObjectValue(
		map[string]attr.Type{
			"description": types.StringType,
		},
		map[string]attr.Value{
			"description": types.StringValue(cluster.Storage.Description),
		},
	)
	diagnostics.Append(storageDiags...)
	data.Storage = storageObj

	data.SharedVolumes = flattenClusterSharedVolumes(cluster.SharedVolumes, diagnostics)
	data.WorkerNodes = flattenClusterWorkerNodes(cluster.WorkerNodes, diagnostics)
}

func flattenClusterSharedVolumes(sharedVolumes []clusterSharedVolume, diagnostics *diag.Diagnostics) types.List {
	if len(sharedVolumes) == 0 {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                types.StringType,
				"name":              types.StringType,
				"mount_point":       types.StringType,
				"size_in_gigabytes": types.Int64Type,
			},
		})
	}

	volumeAttrTypes := map[string]attr.Type{
		"id":                types.StringType,
		"name":              types.StringType,
		"mount_point":       types.StringType,
		"size_in_gigabytes": types.Int64Type,
	}

	var volumeElements []attr.Value
	for _, volume := range sharedVolumes {
		volumeAttrValues := map[string]attr.Value{
			"id":                stringValueOrNull(volume.ID),
			"name":              stringValueOrNull(volume.Name),
			"mount_point":       stringValueOrNull(volume.MountPoint),
			"size_in_gigabytes": int64ValueOrNull(volume.SizeInGigabytes),
		}

		volumeObj, diags := types.ObjectValue(volumeAttrTypes, volumeAttrValues)
		diagnostics.Append(diags...)
		volumeElements = append(volumeElements, volumeObj)
	}

	listValue, diags := types.ListValue(
		types.ObjectType{AttrTypes: volumeAttrTypes},
		volumeElements,
	)
	diagnostics.Append(diags...)
	return listValue
}

func flattenClusterWorkerNodes(workerNodes []clusterWorkerNode, diagnostics *diag.Diagnostics) types.List {
	if len(workerNodes) == 0 {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":         types.StringType,
				"hostname":   types.StringType,
				"status":     types.StringType,
				"private_ip": types.StringType,
				"public_ip":  types.StringType,
			},
		})
	}

	nodeAttrTypes := map[string]attr.Type{
		"id":         types.StringType,
		"hostname":   types.StringType,
		"status":     types.StringType,
		"private_ip": types.StringType,
		"public_ip":  types.StringType,
	}

	var nodeElements []attr.Value
	for _, node := range workerNodes {
		nodeAttrValues := map[string]attr.Value{
			"id":         stringValueOrNull(node.ID),
			"hostname":   stringValueOrNull(node.Hostname),
			"status":     stringValueOrNull(node.Status),
			"private_ip": stringValueOrNull(node.PrivateIP),
			"public_ip":  stringValueOrNull(node.PublicIP),
		}

		nodeObj, diags := types.ObjectValue(nodeAttrTypes, nodeAttrValues)
		diagnostics.Append(diags...)
		nodeElements = append(nodeElements, nodeObj)
	}

	listValue, diags := types.ListValue(
		types.ObjectType{AttrTypes: nodeAttrTypes},
		nodeElements,
	)
	diagnostics.Append(diags...)
	return listValue
}

func stringValueOrNull(value string) attr.Value {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func int64ValueOrNull(value int) attr.Value {
	if value == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(value))
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *verda.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not found") || strings.Contains(errStr, "404")
}
