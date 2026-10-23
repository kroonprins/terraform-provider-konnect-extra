package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &KonnectExtraProvider{}

type KonnectExtraProvider struct {
	version string
}

type KonnectExtraProviderModel struct {
	ServerURL                types.String `tfsdk:"server_url"`
	PersonalAccessToken      types.String `tfsdk:"personal_access_token"`
	SystemAccountAccessToken types.String `tfsdk:"system_account_access_token"`
}

// KonnectExtraClient holds the configured HTTP client and base URL for API calls.
type KonnectExtraClient struct {
	HTTPClient *http.Client
	ServerURL  string
	Token      string
}

func (p *KonnectExtraProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "konnectextra"
	resp.Version = p.version
}

func (p *KonnectExtraProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A supplemental provider for Kong Konnect, providing data sources not yet available in the official provider.",
		Attributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				MarkdownDescription: "The Konnect API server URL. Defaults to `https://global.api.konghq.com`. Configurable via environment variable `KONNECT_SERVER_URL`.",
				Optional:            true,
			},
			"personal_access_token": schema.StringAttribute{
				MarkdownDescription: "A personal access token for authenticating with the Konnect API. Configurable via environment variable `KONNECT_TOKEN`.",
				Optional:            true,
				Sensitive:           true,
			},
			"system_account_access_token": schema.StringAttribute{
				MarkdownDescription: "A system account access token for authenticating with the Konnect API. Configurable via environment variable `KONNECT_SPAT`.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *KonnectExtraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data KonnectExtraProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverURL := data.ServerURL.ValueString()
	if serverURL == "" {
		serverURL = os.Getenv("KONNECT_SERVER_URL")
	}
	if serverURL == "" {
		serverURL = "https://global.api.konghq.com"
	}

	var token string
	if !data.PersonalAccessToken.IsNull() && !data.PersonalAccessToken.IsUnknown() {
		token = data.PersonalAccessToken.ValueString()
	}
	if token == "" {
		token = os.Getenv("KONNECT_TOKEN")
	}
	if token == "" && !data.SystemAccountAccessToken.IsNull() && !data.SystemAccountAccessToken.IsUnknown() {
		token = data.SystemAccountAccessToken.ValueString()
	}
	if token == "" {
		if spat := os.Getenv("KONNECT_SPAT"); spat != "" {
			token = spat
		}
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Authentication",
			"A personal_access_token (KONNECT_TOKEN) or system_account_access_token (KONNECT_SPAT) must be configured.",
		)
		return
	}

	client := &KonnectExtraClient{
		HTTPClient: http.DefaultClient,
		ServerURL:  serverURL,
		Token:      token,
	}

	resp.DataSourceData = client
}

func (p *KonnectExtraProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPortalTeamDataSource,
	}
}

func (p *KonnectExtraProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KonnectExtraProvider{
			version: version,
		}
	}
}
