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

var _ datasource.DataSource = &PortalFormDataSource{}
var _ datasource.DataSourceWithConfigure = &PortalFormDataSource{}

func NewPortalFormDataSource() datasource.DataSource {
	return &PortalFormDataSource{}
}

type PortalFormDataSource struct {
	client *KonnectExtraClient
}

type PortalFormDataSourceModel struct {
	PortalID  types.String `tfsdk:"portal_id"`
	Name      types.String `tfsdk:"name"`
	ID        types.String `tfsdk:"id"`
	Type      types.String `tfsdk:"type"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *PortalFormDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_form"
}

func (d *PortalFormDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a developer portal form by portal ID and name.",
		Attributes: map[string]schema.Attribute{
			"portal_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the portal.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the form to look up.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the form.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The form type (developer_registration or api_registration).",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The publication status of the form.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "An ISO-8601 timestamp of when the form was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "An ISO-8601 timestamp of when the form was last updated.",
			},
		},
	}
}

func (d *PortalFormDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type portalFormAPIResponse struct {
	Data []struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Type      string    `json:"type"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"data"`
}

func (d *PortalFormDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PortalFormDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v3/portals/%s/forms", d.client.ServerURL, url.PathEscape(data.PortalID.ValueString())), nil)
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

	var apiResp portalFormAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error parsing API response", err.Error())
		return
	}
	if len(apiResp.Data) == 0 {
		resp.Diagnostics.AddError("Form not found", fmt.Sprintf("No form with name %q found in portal %q.", data.Name.ValueString(), data.PortalID.ValueString()))
		return
	}
	if len(apiResp.Data) != 1 || apiResp.Data[0].Name != data.Name.ValueString() {
		resp.Diagnostics.AddError("Ambiguous form", fmt.Sprintf("Expected exactly one form with name %q in portal %q.", data.Name.ValueString(), data.PortalID.ValueString()))
		return
	}

	form := apiResp.Data[0]
	data.ID = types.StringValue(form.ID)
	data.Type = types.StringValue(form.Type)
	data.Status = types.StringValue(form.Status)
	data.CreatedAt = types.StringValue(form.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(form.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
