package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &SecretResource{}
var _ resource.ResourceWithImportState = &SecretResource{}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

type SecretResource struct {
	client *verda.Client
}

type SecretResourceModel struct {
	Name       types.String `tfsdk:"name"`
	Value      types.String `tfsdk:"value"`
	SecretType types.String `tfsdk:"secret_type"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

type secretCreateRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type secretResponse struct {
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	SecretType string `json:"secret_type"`
}

func (r *SecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda secret for container and job deployments.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the secret.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Secret value.",
				Required:            true,
				Sensitive:           true,
			},
			"secret_type": schema.StringAttribute{
				MarkdownDescription: "Secret type as reported by the API.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp.",
				Computed:            true,
			},
		},
	}
}

func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := secretCreateRequest{
		Name:  data.Name.ValueString(),
		Value: data.Value.ValueString(),
	}

	if err := doVerdaRequest(ctx, r.client, http.MethodPost, "/secrets", createReq, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create secret, got error: %s", err))
		return
	}

	secret, found, err := r.getSecretByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read secret after creation, got error: %s", err))
		return
	}

	if found {
		data.SecretType = types.StringValue(secret.SecretType)
		data.CreatedAt = types.StringValue(secret.CreatedAt)
	} else {
		resp.Diagnostics.AddWarning("Secret Not Found", "Created secret could not be found in the list")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	priorValue := data.Value

	secret, found, err := r.getSecretByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read secret, got error: %s", err))
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(secret.Name)
	data.SecretType = types.StringValue(secret.SecretType)
	data.CreatedAt = types.StringValue(secret.CreatedAt)
	data.Value = priorValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Secrets cannot be updated. Please delete and recreate the resource.",
	)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretName := url.PathEscape(data.Name.ValueString())
	path := fmt.Sprintf("/secrets/%s", secretName)

	if err := doVerdaRequest(ctx, r.client, http.MethodDelete, path, nil, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete secret, got error: %s", err))
		return
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *SecretResource) getSecretByName(ctx context.Context, name string) (*secretResponse, bool, error) {
	secrets, err := r.listSecrets(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return &secret, true, nil
		}
	}

	return nil, false, nil
}

func (r *SecretResource) listSecrets(ctx context.Context) ([]secretResponse, error) {
	var secrets []secretResponse

	if err := doVerdaRequest(ctx, r.client, http.MethodGet, "/secrets", nil, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}
