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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &StartupScriptBulkDeleteResource{}
var _ resource.ResourceWithImportState = &StartupScriptBulkDeleteResource{}

func NewStartupScriptBulkDeleteResource() resource.Resource {
	return &StartupScriptBulkDeleteResource{}
}

type StartupScriptBulkDeleteResource struct {
	client *verda.Client
}

type StartupScriptBulkDeleteResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ScriptIDs types.List   `tfsdk:"script_ids"`
}

func (r *StartupScriptBulkDeleteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_startup_script_bulk_delete"
}

func (r *StartupScriptBulkDeleteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deletes multiple startup scripts in a single request.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Action identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"script_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Startup script IDs to delete.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *StartupScriptBulkDeleteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StartupScriptBulkDeleteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StartupScriptBulkDeleteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scriptIDs []string
	resp.Diagnostics.Append(data.ScriptIDs.ElementsAs(ctx, &scriptIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.StartupScripts.DeleteMultipleStartupScripts(ctx, scriptIDs); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete startup scripts, got error: %s", err))
		return
	}

	data.ID = types.StringValue(url.PathEscape(strings.Join(scriptIDs, ",")))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StartupScriptBulkDeleteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StartupScriptBulkDeleteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StartupScriptBulkDeleteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Bulk delete must be recreated to run again.")
}

func (r *StartupScriptBulkDeleteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No delete operation for bulk delete actions.
}

func (r *StartupScriptBulkDeleteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
