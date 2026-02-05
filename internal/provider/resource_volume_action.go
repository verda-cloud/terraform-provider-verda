package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/validator/stringvalidator"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &VolumeActionResource{}
var _ resource.ResourceWithImportState = &VolumeActionResource{}

func NewVolumeActionResource() resource.Resource {
	return &VolumeActionResource{}
}

type VolumeActionResource struct {
	client *verda.Client
}

type VolumeActionResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Action       types.String `tfsdk:"action"`
	VolumeIDs    types.List   `tfsdk:"volume_ids"`
	Size         types.Int64  `tfsdk:"size"`
	InstanceID   types.String `tfsdk:"instance_id"`
	InstanceIDs  types.List   `tfsdk:"instance_ids"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	IsPermanent  types.Bool   `tfsdk:"is_permanent"`
	LocationCode types.String `tfsdk:"location_code"`
}

type volumeActionRequest struct {
	Action       string   `json:"action"`
	ID           []string `json:"id"`
	Size         *int     `json:"size,omitempty"`
	InstanceID   string   `json:"instance_id,omitempty"`
	InstanceIDs  []string `json:"instance_ids,omitempty"`
	Name         string   `json:"name,omitempty"`
	Type         string   `json:"type,omitempty"`
	IsPermanent  *bool    `json:"is_permanent,omitempty"`
	LocationCode string   `json:"location_code,omitempty"`
}

func (r *VolumeActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_action"
}

func (r *VolumeActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Performs an action on one or more volumes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Action identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "Action to perform on volumes.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"attach",
						"detach",
						"delete",
						"rename",
						"resize",
						"restore",
						"clone",
						"cancel",
						"create",
						"export",
						"transfer",
					),
				},
			},
			"volume_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Volume IDs to perform the action on.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"size": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "New volume size in GB.",
			},
			"instance_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Instance ID for attach action.",
			},
			"instance_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Instance IDs for attach action.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "New volume name.",
			},
			"type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Target volume type.",
			},
			"is_permanent": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "If deleting, remove the volume permanently.",
			},
			"location_code": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Target location code for clone.",
			},
		},
	}
}

func (r *VolumeActionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VolumeActionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var volumeIDs []string
	resp.Diagnostics.Append(data.VolumeIDs.ElementsAs(ctx, &volumeIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var instanceIDs []string
	if !data.InstanceIDs.IsNull() && !data.InstanceIDs.IsUnknown() {
		resp.Diagnostics.Append(data.InstanceIDs.ElementsAs(ctx, &instanceIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	request := volumeActionRequest{
		Action: data.Action.ValueString(),
		ID:     volumeIDs,
	}

	if !data.Size.IsNull() && !data.Size.IsUnknown() {
		size := int(data.Size.ValueInt64())
		request.Size = &size
	}

	if !data.InstanceID.IsNull() && !data.InstanceID.IsUnknown() {
		request.InstanceID = data.InstanceID.ValueString()
	}

	if len(instanceIDs) > 0 {
		request.InstanceIDs = instanceIDs
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		request.Name = data.Name.ValueString()
	}

	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		request.Type = data.Type.ValueString()
	}

	if !data.IsPermanent.IsNull() && !data.IsPermanent.IsUnknown() {
		value := data.IsPermanent.ValueBool()
		request.IsPermanent = &value
	}

	if !data.LocationCode.IsNull() && !data.LocationCode.IsUnknown() {
		request.LocationCode = data.LocationCode.ValueString()
	}

	if err := doVerdaRequest(ctx, r.client, "PUT", "/volumes", request, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to perform volume action, got error: %s", err))
		return
	}

	idParts := []string{data.Action.ValueString(), strings.Join(volumeIDs, ",")}
	data.ID = types.StringValue(url.PathEscape(strings.Join(idParts, ":")))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VolumeActionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Volume actions must be recreated to run again.")
}

func (r *VolumeActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No delete operation for actions.
}

func (r *VolumeActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
