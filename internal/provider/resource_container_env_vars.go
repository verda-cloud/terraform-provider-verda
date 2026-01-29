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

var _ resource.Resource = &ContainerEnvironmentVariablesResource{}
var _ resource.ResourceWithImportState = &ContainerEnvironmentVariablesResource{}

func NewContainerEnvironmentVariablesResource() resource.Resource {
	return &ContainerEnvironmentVariablesResource{}
}

type ContainerEnvironmentVariablesResource struct {
	client *verda.Client
}

type ContainerEnvironmentVariablesResourceModel struct {
	ID             types.String `tfsdk:"id"`
	DeploymentName types.String `tfsdk:"deployment_name"`
	ContainerName  types.String `tfsdk:"container_name"`
	Env            types.List   `tfsdk:"env"`
}

func (r *ContainerEnvironmentVariablesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_environment_variables"
}

func (r *ContainerEnvironmentVariablesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages environment variables for a container deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier for the environment variables.",
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
			"container_name": schema.StringAttribute{
				MarkdownDescription: "Container name within the deployment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"env": schema.ListNestedAttribute{
				MarkdownDescription: "Environment variables.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "Type of environment variable (plain or secret).",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Environment variable name.",
							Required:            true,
						},
						"value_or_reference_to_secret": schema.StringAttribute{
							MarkdownDescription: "Value or secret reference.",
							Required:            true,
							Sensitive:           true,
						},
					},
				},
			},
		},
	}
}

func (r *ContainerEnvironmentVariablesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContainerEnvironmentVariablesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerEnvironmentVariablesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envVars, err := expandEnvVars(ctx, data.Env)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	request := &verda.EnvironmentVariablesRequest{
		ContainerName: data.ContainerName.ValueString(),
		Env:           envVars,
	}

	if err := r.client.ContainerDeployments.AddEnvironmentVariables(ctx, data.DeploymentName.ValueString(), request); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment variables, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.DeploymentName.ValueString(), data.ContainerName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerEnvironmentVariablesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContainerEnvironmentVariablesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.ContainerDeployments.GetDeploymentByName(ctx, data.DeploymentName.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerEnvironmentVariablesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ContainerEnvironmentVariablesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envVars, err := expandEnvVars(ctx, data.Env)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	request := &verda.EnvironmentVariablesRequest{
		ContainerName: data.ContainerName.ValueString(),
		Env:           envVars,
	}

	if err := r.client.ContainerDeployments.UpdateEnvironmentVariables(ctx, data.DeploymentName.ValueString(), request); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update environment variables, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.DeploymentName.ValueString(), data.ContainerName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerEnvironmentVariablesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ContainerEnvironmentVariablesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envVars, err := expandEnvVars(ctx, data.Env)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	request := &verda.DeleteEnvironmentVariablesRequest{
		ContainerName: data.ContainerName.ValueString(),
		Env:           envVars,
	}

	if err := r.client.ContainerDeployments.DeleteEnvironmentVariables(ctx, data.DeploymentName.ValueString(), request); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment variables, got error: %s", err))
		return
	}
}

func (r *ContainerEnvironmentVariablesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func expandEnvVars(ctx context.Context, envList types.List) ([]verda.ContainerEnvVar, error) {
	var envVars []EnvVarModel
	if err := envList.ElementsAs(ctx, &envVars, false); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %s", err)
	}

	var result []verda.ContainerEnvVar
	for _, envVar := range envVars {
		result = append(result, verda.ContainerEnvVar{
			Type:                     envVar.Type.ValueString(),
			Name:                     envVar.Name.ValueString(),
			ValueOrReferenceToSecret: envVar.ValueOrReferenceToSecret.ValueString(),
		})
	}

	return result, nil
}
