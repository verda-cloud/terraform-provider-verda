package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &FileSecretResource{}
var _ resource.ResourceWithImportState = &FileSecretResource{}

func NewFileSecretResource() resource.Resource {
	return &FileSecretResource{}
}

type FileSecretResource struct {
	client *verda.Client
}

type FileSecretResourceModel struct {
	Name       types.String `tfsdk:"name"`
	Files      types.List   `tfsdk:"files"`
	FileNames  types.List   `tfsdk:"file_names"`
	SecretType types.String `tfsdk:"secret_type"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

type fileSecretFileModel struct {
	FileName      types.String `tfsdk:"file_name"`
	Base64Content types.String `tfsdk:"base64_content"`
}

type fileSecretCreateRequest struct {
	Name  string                  `json:"name"`
	Files []fileSecretFileRequest `json:"files"`
}

type fileSecretFileRequest struct {
	FileName      string `json:"file_name"`
	Base64Content string `json:"base64_content"`
}

type fileSecretResponse struct {
	Name       string   `json:"name"`
	CreatedAt  string   `json:"created_at"`
	SecretType string   `json:"secret_type"`
	FileNames  []string `json:"file_names"`
}

func (r *FileSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_secret"
}

func (r *FileSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda fileset secret for mounting files into deployments.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the file secret.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"files": schema.ListNestedAttribute{
				MarkdownDescription: "Files to store in the secret, base64-encoded.",
				Required:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"file_name": schema.StringAttribute{
							MarkdownDescription: "File name to store.",
							Required:            true,
						},
						"base64_content": schema.StringAttribute{
							MarkdownDescription: "Base64-encoded file content.",
							Required:            true,
							Sensitive:           true,
						},
					},
				},
			},
			"file_names": schema.ListAttribute{
				MarkdownDescription: "File names stored in the secret.",
				ElementType:         types.StringType,
				Computed:            true,
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

func (r *FileSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FileSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var files []fileSecretFileModel
	resp.Diagnostics.Append(data.Files.ElementsAs(ctx, &files, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var requestFiles []fileSecretFileRequest
	for _, file := range files {
		requestFiles = append(requestFiles, fileSecretFileRequest{
			FileName:      file.FileName.ValueString(),
			Base64Content: file.Base64Content.ValueString(),
		})
	}

	createReq := fileSecretCreateRequest{
		Name:  data.Name.ValueString(),
		Files: requestFiles,
	}

	if err := doVerdaRequest(ctx, r.client, http.MethodPost, "/file-secrets", createReq, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create file secret, got error: %s", err))
		return
	}

	secret, found, err := r.getFileSecretByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read file secret after creation, got error: %s", err))
		return
	}

	if found {
		data.SecretType = types.StringValue(secret.SecretType)
		data.CreatedAt = types.StringValue(secret.CreatedAt)
		fileNames, diags := types.ListValueFrom(ctx, types.StringType, secret.FileNames)
		resp.Diagnostics.Append(diags...)
		data.FileNames = fileNames
	} else {
		resp.Diagnostics.AddWarning("File Secret Not Found", "Created file secret could not be found in the list")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	priorFiles := data.Files

	secret, found, err := r.getFileSecretByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read file secret, got error: %s", err))
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(secret.Name)
	data.SecretType = types.StringValue(secret.SecretType)
	data.CreatedAt = types.StringValue(secret.CreatedAt)

	fileNames, diags := types.ListValueFrom(ctx, types.StringType, secret.FileNames)
	resp.Diagnostics.Append(diags...)
	data.FileNames = fileNames
	data.Files = priorFiles

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"File secrets cannot be updated. Please delete and recreate the resource.",
	)
}

func (r *FileSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretName := url.PathEscape(data.Name.ValueString())
	path := fmt.Sprintf("/file-secrets/%s", secretName)

	if err := doVerdaRequest(ctx, r.client, http.MethodDelete, path, nil, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file secret, got error: %s", err))
		return
	}
}

func (r *FileSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *FileSecretResource) getFileSecretByName(ctx context.Context, name string) (*fileSecretResponse, bool, error) {
	secrets, err := r.listFileSecrets(ctx)
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

func (r *FileSecretResource) listFileSecrets(ctx context.Context) ([]fileSecretResponse, error) {
	var secrets []fileSecretResponse

	if err := doVerdaRequest(ctx, r.client, http.MethodGet, "/file-secrets", nil, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}
