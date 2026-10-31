package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PortalFormResource{}
var _ resource.ResourceWithConfigure = &PortalFormResource{}

func NewPortalFormResource() resource.Resource {
	return &PortalFormResource{}
}

type PortalFormResource struct {
	client *KonnectExtraClient
}

type PortalFormResourceModel struct {
	PortalID   types.String `tfsdk:"portal_id"`
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Status     types.String `tfsdk:"status"`
	FieldsJSON types.String `tfsdk:"fields_json"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (r *PortalFormResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_form"
}

func (r *PortalFormResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manage a custom developer portal form.",
		Attributes: map[string]resourceschema.Attribute{
			"portal_id": resourceschema.StringAttribute{
				Required: true, MarkdownDescription: "The ID of the portal.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": resourceschema.StringAttribute{
				Computed: true, MarkdownDescription: "The ID of the form.",
			},
			"name": resourceschema.StringAttribute{
				Required: true, MarkdownDescription: "The form name. Developer registration forms must use `developer-registration`.",
			},
			"type": resourceschema.StringAttribute{
				Required: true, MarkdownDescription: "`developer_registration` or `api_registration`. Cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": resourceschema.StringAttribute{
				Required: true, MarkdownDescription: "`published` or `unpublished`.",
			},
			"fields_json": resourceschema.StringAttribute{
				Required: true, MarkdownDescription: "JSON array of form fields in display order, using the Konnect form field request schema. Include exactly one `submit` field; developer registration also requires `full_name` and `email`, and API registration requires `api_id`.",
			},
			"created_at": resourceschema.StringAttribute{
				Computed: true, MarkdownDescription: "An ISO-8601 timestamp of when the form was created.",
			},
			"updated_at": resourceschema.StringAttribute{
				Computed: true, MarkdownDescription: "An ISO-8601 timestamp of when the form was last updated.",
			},
		},
	}
}

func (r *PortalFormResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*KonnectExtraClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *KonnectExtraClient, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

type portalFormDetail struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Status    string            `json:"status"`
	Fields    []json.RawMessage `json:"fields"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func (r *PortalFormResource) formURL(portalID, formID string) string {
	endpoint := fmt.Sprintf("%s/v3/portals/%s/forms", r.client.ServerURL, url.PathEscape(portalID))
	if formID != "" {
		endpoint += "/" + url.PathEscape(formID)
	}
	return endpoint
}

func (r *PortalFormResource) request(ctx context.Context, method, endpoint string, payload any) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(raw)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return 0, nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+r.client.Token)
	httpReq.Header.Set("Accept", "application/json")
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpResp, err := r.client.HTTPClient.Do(httpReq)
	if err != nil {
		return 0, nil, err
	}
	defer httpResp.Body.Close()
	responseBody, err := io.ReadAll(httpResp.Body)
	return httpResp.StatusCode, responseBody, err
}

func formPayload(data PortalFormResourceModel) (map[string]any, error) {
	if data.Type.ValueString() == "developer_registration" && data.Name.ValueString() != "developer-registration" {
		return nil, fmt.Errorf("developer registration forms must be named developer-registration")
	}
	var fields []json.RawMessage
	if err := json.Unmarshal([]byte(data.FieldsJSON.ValueString()), &fields); err != nil {
		return nil, fmt.Errorf("fields_json must be a JSON array: %w", err)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields_json must contain at least one field")
	}
	return map[string]any{
		"name": data.Name.ValueString(), "type": data.Type.ValueString(),
		"status": data.Status.ValueString(), "fields": fields,
	}, nil
}

func (r *PortalFormResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PortalFormResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, err := formPayload(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid form fields", err.Error())
		return
	}
	status, body, err := r.request(ctx, http.MethodPost, r.formURL(data.PortalID.ValueString(), ""), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error calling Konnect API", err.Error())
		return
	}
	if status != http.StatusCreated {
		resp.Diagnostics.AddError(fmt.Sprintf("Konnect API returned status %d", status), string(body))
		return
	}
	var form portalFormDetail
	if err := json.Unmarshal(body, &form); err != nil || form.ID == "" {
		resp.Diagnostics.AddError("Error parsing API response", fmt.Sprintf("Create response has no valid form ID: %v", err))
		return
	}
	data.ID = types.StringValue(form.ID)
	data.CreatedAt = types.StringValue(form.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(form.UpdatedAt.Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PortalFormResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PortalFormResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status, body, err := r.request(ctx, http.MethodGet, r.formURL(data.PortalID.ValueString(), data.ID.ValueString()), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error calling Konnect API", err.Error())
		return
	}
	if status == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError(fmt.Sprintf("Konnect API returned status %d", status), string(body))
		return
	}
	var form portalFormDetail
	if err := json.Unmarshal(body, &form); err != nil {
		resp.Diagnostics.AddError("Error parsing API response", err.Error())
		return
	}
	fields, err := currentFormFields(data.FieldsJSON.ValueString(), form.Fields)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing form fields", err.Error())
		return
	}
	data.Name = types.StringValue(form.Name)
	data.Type = types.StringValue(form.Type)
	data.Status = types.StringValue(form.Status)
	data.FieldsJSON = types.StringValue(fields)
	data.CreatedAt = types.StringValue(form.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(form.UpdatedAt.Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Keep the configured JSON when the API only adds generated names, defaults, or read-only fields.
func currentFormFields(configured string, actual []json.RawMessage) (string, error) {
	var desired []map[string]any
	if err := json.Unmarshal([]byte(configured), &desired); err != nil {
		return "", err
	}
	var received []map[string]any
	for _, field := range actual {
		var value map[string]any
		if err := json.Unmarshal(field, &value); err != nil {
			return "", err
		}
		delete(value, "built_in")
		received = append(received, value)
	}
	if len(desired) == len(received) {
		equal := true
		for i, field := range desired {
			if !formFieldMatches(field, received[i]) {
				equal = false
			}
		}
		if equal {
			return configured, nil
		}
	}
	raw, err := json.Marshal(received)
	return string(raw), err
}

func formFieldMatches(desired, actual any) bool {
	switch value := desired.(type) {
	case map[string]any:
		found, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for key, item := range value {
			if _, ok := found[key]; !ok || !formFieldMatches(item, found[key]) {
				return false
			}
		}
		return true
	case []any:
		found, ok := actual.([]any)
		if !ok || len(value) != len(found) {
			return false
		}
		for i, item := range value {
			if !formFieldMatches(item, found[i]) {
				return false
			}
		}
		return true
	default:
		return desired == actual
	}
}

func (r *PortalFormResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PortalFormResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior PortalFormResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, err := formPayload(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid form fields", err.Error())
		return
	}
	if data.Type.ValueString() == "developer_registration" {
		delete(payload, "name")
	}
	status, body, err := r.request(ctx, http.MethodPut, r.formURL(data.PortalID.ValueString(), prior.ID.ValueString()), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error calling Konnect API", err.Error())
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError(fmt.Sprintf("Konnect API returned status %d", status), string(body))
		return
	}
	var form portalFormDetail
	if err := json.Unmarshal(body, &form); err != nil {
		resp.Diagnostics.AddError("Error parsing API response", err.Error())
		return
	}
	data.ID = prior.ID
	data.CreatedAt = types.StringValue(form.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(form.UpdatedAt.Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PortalFormResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PortalFormResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status, body, err := r.request(ctx, http.MethodDelete, r.formURL(data.PortalID.ValueString(), data.ID.ValueString()), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error calling Konnect API", err.Error())
		return
	}
	if status != http.StatusNoContent && status != http.StatusNotFound {
		resp.Diagnostics.AddError(fmt.Sprintf("Konnect API returned status %d", status), string(body))
	}
}
