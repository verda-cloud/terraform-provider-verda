package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &StartupScriptResource{}
var _ resource.ResourceWithImportState = &StartupScriptResource{}

func NewStartupScriptResource() resource.Resource {
	return &StartupScriptResource{}
}

type StartupScriptResource struct {
	client *verda.Client
}

type StartupScriptResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Script    types.String `tfsdk:"script"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *StartupScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_startup_script"
}

func (r *StartupScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda startup script",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Startup script identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the startup script",
				Required:            true,
			},
			"script": schema.StringAttribute{
				MarkdownDescription: "Script content to execute on instance startup",
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
			},
		},
	}
}

func (r *StartupScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StartupScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StartupScriptResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := verda.CreateStartupScriptRequest{
		Name:   data.Name.ValueString(),
		Script: data.Script.ValueString(),
	}

	script, err := r.client.StartupScripts.AddStartupScript(ctx, &createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create startup script, got error: %s", err))
		return
	}

	data.ID = types.StringValue(script.ID)
	data.CreatedAt = types.StringValue(script.CreatedAt.Format("2006-01-02T15:04:05Z"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StartupScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StartupScriptResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	script, err := r.client.StartupScripts.GetStartupScriptByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read startup script, got error: %s", err))
		return
	}

	data.Name = types.StringValue(script.Name)
	data.Script = types.StringValue(script.Script)
	data.CreatedAt = types.StringValue(script.CreatedAt.Format("2006-01-02T15:04:05Z"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StartupScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StartupScriptResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Startup scripts cannot be updated in the Verda API, only deleted and recreated
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Startup scripts cannot be updated. Please delete and recreate the resource.",
	)
}

func (r *StartupScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StartupScriptResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.StartupScripts.DeleteStartupScript(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete startup script, got error: %s", err))
		return
	}
}

func (r *StartupScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
