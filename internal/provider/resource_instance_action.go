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

var _ resource.Resource = &InstanceActionResource{}
var _ resource.ResourceWithImportState = &InstanceActionResource{}

func NewInstanceActionResource() resource.Resource {
	return &InstanceActionResource{}
}

type InstanceActionResource struct {
	client *verda.Client
}

type InstanceActionResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Action                types.String `tfsdk:"action"`
	InstanceIDs           types.List   `tfsdk:"instance_ids"`
	VolumeIDs             types.List   `tfsdk:"volume_ids"`
	UseDeprecatedEndpoint types.Bool   `tfsdk:"use_deprecated_endpoint"`
}

type instanceActionRequest struct {
	ID        []string `json:"id"`
	Action    string   `json:"action"`
	VolumeIDs []string `json:"volume_ids,omitempty"`
}

func (r *InstanceActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_action"
}

func (r *InstanceActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Performs an action on one or more instances.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Action identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "Action to perform on instances.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"boot",
						"start",
						"shutdown",
						"delete",
						"discontinue",
						"hibernate",
						"configure_spot",
						"force_shutdown",
						"delete_stuck",
						"deploy",
						"transfer",
					),
				},
			},
			"instance_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Instance IDs to perform the action on.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"volume_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Volume IDs to delete when action is delete/delete_stuck.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"use_deprecated_endpoint": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Use deprecated /instances/action endpoint.",
			},
		},
	}
}

func (r *InstanceActionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InstanceActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InstanceActionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var instanceIDs []string
	resp.Diagnostics.Append(data.InstanceIDs.ElementsAs(ctx, &instanceIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var volumeIDs []string
	if !data.VolumeIDs.IsNull() && !data.VolumeIDs.IsUnknown() {
		resp.Diagnostics.Append(data.VolumeIDs.ElementsAs(ctx, &volumeIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	request := instanceActionRequest{
		ID:        instanceIDs,
		Action:    data.Action.ValueString(),
		VolumeIDs: volumeIDs,
	}

	method := "PUT"
	path := "/instances"
	if !data.UseDeprecatedEndpoint.IsNull() && data.UseDeprecatedEndpoint.ValueBool() {
		path = "/instances/action"
		method = "POST"
	}

	if err := doVerdaRequest(ctx, r.client, method, path, request, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to perform instance action, got error: %s", err))
		return
	}

	idParts := []string{data.Action.ValueString(), strings.Join(instanceIDs, ",")}
	data.ID = types.StringValue(url.PathEscape(strings.Join(idParts, ":")))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InstanceActionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Instance actions must be recreated to run again.")
}

func (r *InstanceActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No delete operation for actions.
}

func (r *InstanceActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
