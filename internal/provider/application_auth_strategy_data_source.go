package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ApplicationAuthStrategyDataSource{}
var _ datasource.DataSourceWithConfigure = &ApplicationAuthStrategyDataSource{}

func NewApplicationAuthStrategyDataSource() datasource.DataSource {
	return &ApplicationAuthStrategyDataSource{}
}

type ApplicationAuthStrategyDataSource struct {
	client *KonnectExtraClient
}

type ApplicationAuthStrategyDataSourceModel struct {
	Name                        types.String `tfsdk:"name"`
	ID                          types.String `tfsdk:"id"`
	DisplayName                 types.String `tfsdk:"display_name"`
	StrategyType                types.String `tfsdk:"strategy_type"`
	Active                      types.Bool   `tfsdk:"active"`
	DCRProviderID               types.String `tfsdk:"dcr_provider_id"`
	SupportsMultipleCredentials types.Bool   `tfsdk:"supports_multiple_credentials"`
	CreatedAt                   types.String `tfsdk:"created_at"`
	UpdatedAt                   types.String `tfsdk:"updated_at"`
}

func (d *ApplicationAuthStrategyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_auth_strategy"
}

func (d *ApplicationAuthStrategyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an application auth strategy by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the application auth strategy.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the application auth strategy.",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The display name shown in the portal.",
			},
			"strategy_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The strategy type (key_auth or openid_connect).",
			},
			"active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether a published entity uses this strategy.",
			},
			"dcr_provider_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the associated DCR provider, if any.",
			},
			"supports_multiple_credentials": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this strategy supports multiple credentials.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "An ISO-8601 timestamp of when the strategy was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "An ISO-8601 timestamp of when the strategy was last updated.",
			},
		},
	}
}

func (d *ApplicationAuthStrategyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*KonnectExtraClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *KonnectExtraClient, got: %T.", req.ProviderData))
		return
	}
	d.client = client
}

type applicationAuthStrategyAPIResponse struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		DisplayName  string `json:"display_name"`
		StrategyType string `json:"strategy_type"`
		Active       *bool  `json:"active"`
		DCRProvider  *struct {
			ID string `json:"id"`
		} `json:"dcr_provider"`
		SupportsMultipleCredentials *bool     `json:"supports_multiple_credentials"`
		CreatedAt                   time.Time `json:"created_at"`
		UpdatedAt                   time.Time `json:"updated_at"`
	} `json:"data"`
}

func (d *ApplicationAuthStrategyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationAuthStrategyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, d.client.ServerURL+"/v2/application-auth-strategies", nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}
	q := httpReq.URL.Query()
	q.Set("filter[name][eq]", data.Name.ValueString())
	httpReq.URL.RawQuery = q.Encode()
	httpReq.Header.Set("Authorization", "Bearer "+d.client.Token)
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := d.client.HTTPClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error calling Konnect API", err.Error())
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading response body", err.Error())
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(fmt.Sprintf("Konnect API returned status %d", httpResp.StatusCode), string(body))
		return
	}

	var apiResp applicationAuthStrategyAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error parsing API response", err.Error())
		return
	}
	if len(apiResp.Data) == 0 {
		resp.Diagnostics.AddError("Auth strategy not found", fmt.Sprintf("No application auth strategy with name %q found.", data.Name.ValueString()))
		return
	}
	if len(apiResp.Data) != 1 || apiResp.Data[0].Name != data.Name.ValueString() {
		resp.Diagnostics.AddError("Ambiguous auth strategy", fmt.Sprintf("Expected exactly one application auth strategy with name %q.", data.Name.ValueString()))
		return
	}

	strategy := apiResp.Data[0]
	data.ID = types.StringValue(strategy.ID)
	data.DisplayName = types.StringValue(strategy.DisplayName)
	data.StrategyType = types.StringValue(strategy.StrategyType)
	data.Active = types.BoolPointerValue(strategy.Active)
	if strategy.DCRProvider != nil {
		data.DCRProviderID = types.StringValue(strategy.DCRProvider.ID)
	} else {
		data.DCRProviderID = types.StringNull()
	}
	data.SupportsMultipleCredentials = types.BoolPointerValue(strategy.SupportsMultipleCredentials)
	data.CreatedAt = types.StringValue(strategy.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(strategy.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
