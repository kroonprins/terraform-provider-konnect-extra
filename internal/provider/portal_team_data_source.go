package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PortalTeamDataSource{}
var _ datasource.DataSourceWithConfigure = &PortalTeamDataSource{}

func NewPortalTeamDataSource() datasource.DataSource {
	return &PortalTeamDataSource{}
}

type PortalTeamDataSource struct {
	client *KonnectExtraClient
}

type PortalTeamDataSourceModel struct {
	PortalID           types.String `tfsdk:"portal_id"`
	Name               types.String `tfsdk:"name"`
	ID                 types.String `tfsdk:"id"`
	Description        types.String `tfsdk:"description"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	CanOwnApplications types.Bool   `tfsdk:"can_own_applications"`
	KonnectManaged     types.Bool   `tfsdk:"konnect_managed"`
}

func (d *PortalTeamDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_team"
}

func (d *PortalTeamDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a portal team by portal ID and team name.",
		Attributes: map[string]schema.Attribute{
			"portal_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the portal.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the team to look up.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the team.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the team.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "An ISO-8601 timestamp of when the team was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "An ISO-8601 timestamp of when the team was last updated.",
			},
			"can_own_applications": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the team is allowed to own applications.",
			},
			"konnect_managed": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the team's membership is managed by Konnect instead of being synced from an identity provider's team mappings.",
			},
		},
	}
}

func (d *PortalTeamDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*KonnectExtraClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *KonnectExtraClient, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// portalTeamAPIResponse matches the JSON structure returned by the Konnect API.
type portalTeamAPIResponse struct {
	Data []portalTeamAPIItem `json:"data"`
}

type portalTeamAPIItem struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CanOwnApplications *bool     `json:"can_own_applications"`
	KonnectManaged     *bool     `json:"konnect_managed"`
}

func (d *PortalTeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PortalTeamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	portalID := data.PortalID.ValueString()
	teamName := data.Name.ValueString()

	apiURL := fmt.Sprintf("%s/v3/portals/%s/teams", d.client.ServerURL, url.PathEscape(portalID))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}

	q := httpReq.URL.Query()
	q.Set("filter[name][eq]", teamName)
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
		resp.Diagnostics.AddError(
			fmt.Sprintf("Konnect API returned status %d", httpResp.StatusCode),
			string(body),
		)
		return
	}

	var apiResp portalTeamAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error parsing API response", err.Error())
		return
	}

	if len(apiResp.Data) == 0 {
		resp.Diagnostics.AddError(
			"Team not found",
			fmt.Sprintf("No team with name %q found in portal %q.", teamName, portalID),
		)
		return
	}

	team := apiResp.Data[0]
	data.ID = types.StringValue(team.ID)
	data.Name = types.StringValue(team.Name)
	data.Description = types.StringValue(team.Description)
	data.CreatedAt = types.StringValue(team.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(team.UpdatedAt.Format(time.RFC3339))
	data.CanOwnApplications = types.BoolPointerValue(team.CanOwnApplications)
	data.KonnectManaged = types.BoolPointerValue(team.KonnectManaged)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
