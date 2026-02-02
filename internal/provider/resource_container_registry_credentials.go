package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &ContainerRegistryCredentialsResource{}
var _ resource.ResourceWithImportState = &ContainerRegistryCredentialsResource{}

func NewContainerRegistryCredentialsResource() resource.Resource {
	return &ContainerRegistryCredentialsResource{}
}

type ContainerRegistryCredentialsResource struct {
	client *verda.Client
}

type ContainerRegistryCredentialsResourceModel struct {
	Name              types.String `tfsdk:"name"`
	Type              types.String `tfsdk:"type"`
	Username          types.String `tfsdk:"username"`
	AccessToken       types.String `tfsdk:"access_token"`
	ServiceAccountKey types.String `tfsdk:"service_account_key"`
	DockerConfigJSON  types.String `tfsdk:"docker_config_json"`
	AccessKeyID       types.String `tfsdk:"access_key_id"`
	SecretAccessKey   types.String `tfsdk:"secret_access_key"`
	Region            types.String `tfsdk:"region"`
	EcrRepo           types.String `tfsdk:"ecr_repo"`
	ScalewayDomain    types.String `tfsdk:"scaleway_domain"`
	ScalewayUUID      types.String `tfsdk:"scaleway_uuid"`
	CreatedAt         types.String `tfsdk:"created_at"`
}

func (r *ContainerRegistryCredentialsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_credentials"
}

func (r *ContainerRegistryCredentialsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages container registry credentials for accessing private container registries",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the registry credentials",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of registry: 'dockerhub', 'gcr', 'ghcr', 'ecr', 'scaleway', etc.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for registry authentication (for DockerHub, GHCR, etc.)",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access token for registry authentication (for DockerHub, GHCR, etc.)",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_account_key": schema.StringAttribute{
				MarkdownDescription: "Service account key JSON for GCR authentication",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"docker_config_json": schema.StringAttribute{
				MarkdownDescription: "Docker config.json content for authentication",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_key_id": schema.StringAttribute{
				MarkdownDescription: "AWS Access Key ID for ECR authentication",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_access_key": schema.StringAttribute{
				MarkdownDescription: "AWS Secret Access Key for ECR authentication",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region for ECR",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ecr_repo": schema.StringAttribute{
				MarkdownDescription: "ECR repository URL",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scaleway_domain": schema.StringAttribute{
				MarkdownDescription: "Scaleway registry domain",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scaleway_uuid": schema.StringAttribute{
				MarkdownDescription: "Scaleway namespace UUID",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the credentials were created",
				Computed:            true,
			},
		},
	}
}

func (r *ContainerRegistryCredentialsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContainerRegistryCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerRegistryCredentialsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &verda.CreateRegistryCredentialsRequest{
		Name: data.Name.ValueString(),
		Type: data.Type.ValueString(),
	}

	if !data.Username.IsNull() {
		createReq.Username = data.Username.ValueString()
	}

	if !data.AccessToken.IsNull() {
		createReq.AccessToken = data.AccessToken.ValueString()
	}

	if !data.ServiceAccountKey.IsNull() {
		createReq.ServiceAccountKey = data.ServiceAccountKey.ValueString()
	}

	if !data.DockerConfigJSON.IsNull() {
		createReq.DockerConfigJson = data.DockerConfigJSON.ValueString()
	}

	if !data.AccessKeyID.IsNull() {
		createReq.AccessKeyID = data.AccessKeyID.ValueString()
	}

	if !data.SecretAccessKey.IsNull() {
		createReq.SecretAccessKey = data.SecretAccessKey.ValueString()
	}

	if !data.Region.IsNull() {
		createReq.Region = data.Region.ValueString()
	}

	if !data.EcrRepo.IsNull() {
		createReq.EcrRepo = data.EcrRepo.ValueString()
	}

	if !data.ScalewayDomain.IsNull() {
		createReq.ScalewayDomain = data.ScalewayDomain.ValueString()
	}

	if !data.ScalewayUUID.IsNull() {
		createReq.ScalewayUUID = data.ScalewayUUID.ValueString()
	}

	err := r.client.ContainerDeployments.CreateRegistryCredentials(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create registry credentials, got error: %s", err))
		return
	}

	// The API doesn't return the created credentials, so we need to fetch them
	credentials, err := r.client.ContainerDeployments.GetRegistryCredentials(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read registry credentials after creation, got error: %s", err))
		return
	}

	// Find our newly created credentials
	var found bool
	for _, cred := range credentials {
		if cred.Name == data.Name.ValueString() {
			data.CreatedAt = types.StringValue(cred.CreatedAt.Format(time.RFC3339))
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddWarning("Credentials Not Found", "Created credentials could not be found in the list")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryCredentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContainerRegistryCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	credentials, err := r.client.ContainerDeployments.GetRegistryCredentials(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read registry credentials, got error: %s", err))
		return
	}

	// Find our credentials by name
	var found bool
	for _, cred := range credentials {
		if cred.Name == data.Name.ValueString() {
			data.CreatedAt = types.StringValue(cred.CreatedAt.Format(time.RFC3339))
			found = true
			break
		}
	}

	if !found {
		// Credentials no longer exist
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryCredentialsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ContainerRegistryCredentialsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Registry credentials cannot be updated, only deleted and recreated
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Registry credentials cannot be updated. Please delete and recreate the resource with new values.",
	)
}

func (r *ContainerRegistryCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ContainerRegistryCredentialsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ContainerDeployments.DeleteRegistryCredentials(ctx, data.Name.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete registry credentials, got error: %s", err))
		return
	}
}

func (r *ContainerRegistryCredentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
