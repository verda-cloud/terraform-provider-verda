package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/validator/stringvalidator"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &ContainerActionResource{}
var _ resource.ResourceWithImportState = &ContainerActionResource{}

func NewContainerActionResource() resource.Resource {
	return &ContainerActionResource{}
}

type ContainerActionResource struct {
	client *verda.Client
}

type ContainerActionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	DeploymentName types.String `tfsdk:"deployment_name"`
	Action         types.String `tfsdk:"action"`
}

func (r *ContainerActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_action"
}

func (r *ContainerActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Performs an action on a container deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Action identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_name": schema.StringAttribute{
				MarkdownDescription: "Deployment name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "Action to perform: pause, resume, restart, or purge_queue.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("pause", "resume", "restart", "purge_queue"),
				},
			},
		},
	}
}

func (r *ContainerActionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContainerActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerActionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	switch data.Action.ValueString() {
	case "pause":
		err = r.client.ContainerDeployments.PauseDeployment(ctx, data.DeploymentName.ValueString())
	case "resume":
		err = r.client.ContainerDeployments.ResumeDeployment(ctx, data.DeploymentName.ValueString())
	case "restart":
		err = r.client.ContainerDeployments.RestartDeployment(ctx, data.DeploymentName.ValueString())
	case "purge_queue":
		err = r.client.ContainerDeployments.PurgeDeploymentQueue(ctx, data.DeploymentName.ValueString())
	default:
		err = fmt.Errorf("unsupported action: %s", data.Action.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to perform container action, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.DeploymentName.ValueString(), data.Action.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContainerActionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Container actions must be recreated to run again.")
}

func (r *ContainerActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No delete operation for actions.
}

func (r *ContainerActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
