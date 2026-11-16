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

type lookupDataSource struct {
	client *KonnectExtraClient
}

func (d *lookupDataSource) configure(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *lookupDataSource) get(ctx context.Context, path string, query url.Values, destination any) error {
	apiURL, err := url.JoinPath(d.client.ServerURL, path)
	if err != nil {
		return fmt.Errorf("building API URL: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	httpReq.URL.RawQuery = query.Encode()
	httpReq.Header.Set("Authorization", "Bearer "+d.client.Token)
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := d.client.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("calling Konnect API: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("Konnect API returned status %d: %s", httpResp.StatusCode, body)
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("parsing API response: %w", err)
	}
	return nil
}

type portalDeveloperAPIResponse struct {
	Data []portalDeveloperAPIItem `json:"data"`
}

type portalDeveloperAPIItem struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PortalDeveloperDataSource struct{ lookupDataSource }

type PortalDeveloperDataSourceModel struct {
	PortalID  types.String `tfsdk:"portal_id"`
	Name      types.String `tfsdk:"name"`
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	FullName  types.String `tfsdk:"full_name"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewPortalDeveloperDataSource() datasource.DataSource { return &PortalDeveloperDataSource{} }

func (d *PortalDeveloperDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_developer"
}

func (d *PortalDeveloperDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a portal developer by portal ID and full name.",
		Attributes: map[string]schema.Attribute{
			"portal_id":  requiredStringAttribute("The ID of the portal."),
			"name":       requiredStringAttribute("The full name of the developer to look up."),
			"id":         computedStringAttribute("The ID of the developer."),
			"email":      computedStringAttribute("The email address of the developer."),
			"full_name":  computedStringAttribute("The full name of the developer."),
			"status":     computedStringAttribute("The status of the developer."),
			"created_at": computedStringAttribute("An ISO-8601 timestamp of when the developer was created."),
			"updated_at": computedStringAttribute("An ISO-8601 timestamp of when the developer was last updated."),
		},
	}
}

func (d *PortalDeveloperDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}

func (d *PortalDeveloperDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PortalDeveloperDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp portalDeveloperAPIResponse
	query := url.Values{"filter[name][eq]": []string{data.Name.ValueString()}}
	if err := d.get(ctx, fmt.Sprintf("/v3/portals/%s/developers", url.PathEscape(data.PortalID.ValueString())), query, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error looking up portal developer", err.Error())
		return
	}
	if len(apiResp.Data) != 1 || apiResp.Data[0].FullName != data.Name.ValueString() {
		resp.Diagnostics.AddError("Portal developer not found or ambiguous", fmt.Sprintf("Expected exactly one developer with name %q in portal %q.", data.Name.ValueString(), data.PortalID.ValueString()))
		return
	}

	developer := apiResp.Data[0]
	data.ID = types.StringValue(developer.ID)
	data.Email = types.StringValue(developer.Email)
	data.FullName = types.StringValue(developer.FullName)
	data.Status = types.StringValue(developer.Status)
	data.CreatedAt = types.StringValue(developer.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(developer.UpdatedAt.Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type portalApplicationAPIResponse struct {
	Data []portalApplicationAPIItem `json:"data"`
}

type portalApplicationAPIItem struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description"`
	ClientID          *string   `json:"client_id"`
	RegistrationCount float64   `json:"registration_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	AuthStrategy      struct {
		ID string `json:"id"`
	} `json:"auth_strategy"`
}

type PortalApplicationDataSource struct{ lookupDataSource }

type PortalApplicationDataSourceModel struct {
	PortalID          types.String                           `tfsdk:"portal_id"`
	Name              types.String                           `tfsdk:"name"`
	Owner             *PortalApplicationOwnerDataSourceModel `tfsdk:"owner"`
	ID                types.String                           `tfsdk:"id"`
	Description       types.String                           `tfsdk:"description"`
	ClientID          types.String                           `tfsdk:"client_id"`
	AuthStrategyID    types.String                           `tfsdk:"auth_strategy_id"`
	RegistrationCount types.Float64                          `tfsdk:"registration_count"`
	CreatedAt         types.String                           `tfsdk:"created_at"`
	UpdatedAt         types.String                           `tfsdk:"updated_at"`
}

type PortalApplicationOwnerDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

func NewPortalApplicationDataSource() datasource.DataSource { return &PortalApplicationDataSource{} }

func (d *PortalApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_application"
}

func (d *PortalApplicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a portal application by portal ID and name.",
		Attributes: map[string]schema.Attribute{
			"portal_id": requiredStringAttribute("The ID of the portal."),
			"name":      requiredStringAttribute("The name of the application to look up."),
			"owner": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Optionally restrict the lookup to an application owned by a developer or team.",
				Attributes: map[string]schema.Attribute{
					"id":   requiredStringAttribute("The owner ID. A developer ID when type is developer, or a team ID when type is team."),
					"type": requiredStringAttribute("The owner type: developer or team."),
				},
			},
			"id":                 computedStringAttribute("The ID of the application."),
			"description":        computedStringAttribute("The description of the application."),
			"client_id":          computedStringAttribute("The identity provider client ID, if any."),
			"auth_strategy_id":   computedStringAttribute("The ID of the application's auth strategy."),
			"registration_count": schema.Float64Attribute{Computed: true, MarkdownDescription: "The number of API registrations associated with the application."},
			"created_at":         computedStringAttribute("An ISO-8601 timestamp of when the application was created."),
			"updated_at":         computedStringAttribute("An ISO-8601 timestamp of when the application was last updated."),
		},
	}
}

func (d *PortalApplicationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}

func (d *PortalApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PortalApplicationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp portalApplicationAPIResponse
	query := url.Values{"filter[name][eq]": []string{data.Name.ValueString()}}
	if data.Owner != nil {
		switch data.Owner.Type.ValueString() {
		case "developer":
			query.Set("filter[developer_id][eq]", data.Owner.ID.ValueString())
		case "team":
			query.Set("filter[team_id][eq]", data.Owner.ID.ValueString())
		default:
			resp.Diagnostics.AddError(
				"Invalid portal application owner type",
				fmt.Sprintf("Owner type must be %q or %q, got %q.", "developer", "team", data.Owner.Type.ValueString()),
			)
			return
		}
	}
	if err := d.get(ctx, fmt.Sprintf("/v3/portals/%s/applications", url.PathEscape(data.PortalID.ValueString())), query, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error looking up portal application", err.Error())
		return
	}
	if len(apiResp.Data) != 1 || apiResp.Data[0].Name != data.Name.ValueString() {
		resp.Diagnostics.AddError("Portal application not found or ambiguous", fmt.Sprintf("Expected exactly one application with name %q in portal %q.", data.Name.ValueString(), data.PortalID.ValueString()))
		return
	}

	application := apiResp.Data[0]
	data.ID = types.StringValue(application.ID)
	data.Description = types.StringPointerValue(application.Description)
	data.ClientID = types.StringPointerValue(application.ClientID)
	data.AuthStrategyID = types.StringValue(application.AuthStrategy.ID)
	data.RegistrationCount = types.Float64Value(application.RegistrationCount)
	data.CreatedAt = types.StringValue(application.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(application.UpdatedAt.Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type apiListResponse struct {
	Data []apiAPIItem `json:"data"`
	Meta struct {
		Page struct {
			Number int `json:"number"`
			Size   int `json:"size"`
			Total  int `json:"total"`
		} `json:"page"`
	} `json:"meta"`
}

type apiAPIItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     *string   `json:"version"`
	Description *string   `json:"description"`
	Slug        *string   `json:"slug"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type APIDataSourceAPIModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Slug        types.String `tfsdk:"slug"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func apiAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":          computedStringAttribute("The ID of the API."),
		"name":        computedStringAttribute("The name of the API."),
		"version":     computedStringAttribute("The version of the API, if any."),
		"description": computedStringAttribute("The description of the API."),
		"slug":        computedStringAttribute("The URL slug of the API."),
		"created_at":  computedStringAttribute("An ISO-8601 timestamp of when the API was created."),
		"updated_at":  computedStringAttribute("An ISO-8601 timestamp of when the API was last updated."),
	}
}

func apiModel(api apiAPIItem) APIDataSourceAPIModel {
	return APIDataSourceAPIModel{
		ID:          types.StringValue(api.ID),
		Name:        types.StringValue(api.Name),
		Version:     types.StringPointerValue(api.Version),
		Description: types.StringPointerValue(api.Description),
		Slug:        types.StringPointerValue(api.Slug),
		CreatedAt:   types.StringValue(api.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:   types.StringValue(api.UpdatedAt.Format(time.RFC3339)),
	}
}

func (d *lookupDataSource) listAPIs(ctx context.Context, name string, version *string) ([]apiAPIItem, error) {
	query := url.Values{
		"filter[name][eq]": []string{name},
		"page[size]":       []string{"100"},
	}
	if version != nil {
		query.Set("filter[version][eq]", *version)
	}
	var apis []apiAPIItem
	for page := 1; ; page++ {
		query.Set("page[number]", fmt.Sprintf("%d", page))
		var response apiListResponse
		if err := d.get(ctx, "/v3/apis", query, &response); err != nil {
			return nil, err
		}
		apis = append(apis, response.Data...)
		if response.Meta.Page.Total <= len(apis) || len(response.Data) == 0 {
			return apis, nil
		}
	}
}

type APIsDataSource struct{ lookupDataSource }

type APIsDataSourceModel struct {
	Name types.String            `tfsdk:"name"`
	APIs []APIDataSourceAPIModel `tfsdk:"apis"`
}

func NewAPIsDataSource() datasource.DataSource { return &APIsDataSource{} }

func (d *APIsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apis"
}

func (d *APIsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List catalog APIs by name.",
		Attributes: map[string]schema.Attribute{
			"name": requiredStringAttribute("The name of the APIs to list."),
			"apis": schema.ListNestedAttribute{Computed: true, MarkdownDescription: "The APIs with the requested name.", NestedObject: schema.NestedAttributeObject{Attributes: apiAttributes()}},
		},
	}
}

func (d *APIsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}

func (d *APIsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apis, err := d.listAPIs(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error listing catalog APIs", err.Error())
		return
	}
	data.APIs = make([]APIDataSourceAPIModel, len(apis))
	for i, api := range apis {
		data.APIs[i] = apiModel(api)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type APIByNameDataSource struct{ lookupDataSource }

type APIByNameDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Slug        types.String `tfsdk:"slug"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewAPIByNameDataSource() datasource.DataSource { return &APIByNameDataSource{} }

func (d *APIByNameDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_by_name"
}

func (d *APIByNameDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := apiAttributes()
	attributes["name"] = requiredStringAttribute("The name of the API to look up.")
	resp.Schema = schema.Schema{MarkdownDescription: "Look up exactly one catalog API by name.", Attributes: attributes}
}

func (d *APIByNameDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}

func (d *APIByNameDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIByNameDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apis, err := d.listAPIs(ctx, data.Name.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error looking up catalog API", err.Error())
		return
	}
	if len(apis) != 1 || apis[0].Name != data.Name.ValueString() {
		resp.Diagnostics.AddError("Catalog API not found or ambiguous", fmt.Sprintf("Expected exactly one API with name %q.", data.Name.ValueString()))
		return
	}
	api := apiModel(apis[0])
	data.ID = api.ID
	data.Version = api.Version
	data.Description = api.Description
	data.Slug = api.Slug
	data.CreatedAt = api.CreatedAt
	data.UpdatedAt = api.UpdatedAt
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type APIDataSource struct{ lookupDataSource }

type APIDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Slug        types.String `tfsdk:"slug"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewAPIDataSource() datasource.DataSource { return &APIDataSource{} }

func (d *APIDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api"
}

func (d *APIDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := apiAttributes()
	attributes["name"] = requiredStringAttribute("The name of the API to look up.")
	attributes["version"] = requiredStringAttribute("The version of the API to look up.")
	resp.Schema = schema.Schema{MarkdownDescription: "Look up a catalog API by name and version.", Attributes: attributes}
}

func (d *APIDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}

func (d *APIDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	version := data.Version.ValueString()
	apis, err := d.listAPIs(ctx, data.Name.ValueString(), &version)
	if err != nil {
		resp.Diagnostics.AddError("Error looking up catalog API", err.Error())
		return
	}
	if len(apis) != 1 || apis[0].Name != data.Name.ValueString() ||
		apis[0].Version == nil || *apis[0].Version != data.Version.ValueString() {
		resp.Diagnostics.AddError("Catalog API not found or ambiguous", fmt.Sprintf("Expected exactly one API with name %q and version %q.", data.Name.ValueString(), data.Version.ValueString()))
		return
	}
	api := apiModel(apis[0])
	data.ID = api.ID
	data.Description = api.Description
	data.Slug = api.Slug
	data.CreatedAt = api.CreatedAt
	data.UpdatedAt = api.UpdatedAt
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func requiredStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{Required: true, MarkdownDescription: description}
}

func computedStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{Computed: true, MarkdownDescription: description}
}
