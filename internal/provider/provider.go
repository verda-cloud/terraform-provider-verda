package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ provider.Provider = &VerdaProvider{}

type VerdaProvider struct {
	version string
}

type VerdaProviderModel struct {
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	BaseURL      types.String `tfsdk:"base_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &VerdaProvider{
			version: version,
		}
	}
}

func (p *VerdaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "verda"
	resp.Version = p.version
}

func (p *VerdaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Verda OAuth2 Client ID. Can also be set via VERDA_CLIENT_ID environment variable.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Verda OAuth2 Client Secret. Can also be set via VERDA_CLIENT_SECRET environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Verda API Base URL. Defaults to https://api.verda.com/v1. Can also be set via VERDA_BASE_URL environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *VerdaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data VerdaProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	clientID := os.Getenv("VERDA_CLIENT_ID")
	clientSecret := os.Getenv("VERDA_CLIENT_SECRET")
	baseURL := os.Getenv("VERDA_BASE_URL")

	if !data.ClientID.IsNull() {
		clientID = data.ClientID.ValueString()
	}

	if !data.ClientSecret.IsNull() {
		clientSecret = data.ClientSecret.ValueString()
	}

	if !data.BaseURL.IsNull() {
		baseURL = data.BaseURL.ValueString()
	}

	if clientID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Missing Verda API Client ID",
			"The provider cannot create the Verda API client as there is a missing or empty value for the Verda API client ID. "+
				"Set the client_id value in the configuration or use the VERDA_CLIENT_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if clientSecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Missing Verda API Client Secret",
			"The provider cannot create the Verda API client as there is a missing or empty value for the Verda API client secret. "+
				"Set the client_secret value in the configuration or use the VERDA_CLIENT_SECRET environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Construct User-Agent: verda-terraform-provider/{provider_version}(terraform/{terraform_version})
	userAgent := fmt.Sprintf("verda-terraform-provider/%s(terraform/%s)", p.version, req.TerraformVersion)

	opts := []verda.ClientOption{
		verda.WithClientID(clientID),
		verda.WithClientSecret(clientSecret),
		verda.WithUserAgent(userAgent),
	}

	if baseURL != "" {
		opts = append(opts, verda.WithBaseURL(baseURL))
	}

	client, err := verda.NewClient(opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Verda API Client",
			"An unexpected error occurred when creating the Verda API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Verda Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *VerdaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewSSHKeyResource,
		NewStartupScriptResource,
		NewVolumeResource,
		NewContainerResource,
		NewContainerRegistryCredentialsResource,
		NewServerlessJobResource,
	}
}

func (p *VerdaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
