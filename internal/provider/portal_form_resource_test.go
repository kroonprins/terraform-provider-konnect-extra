package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func formPlan(t *testing.T, schema resource.SchemaResponse, values map[string]string) tfsdk.Plan {
	t.Helper()
	objectType := schema.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	raw := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, typ := range objectType.AttributeTypes {
		if value, ok := values[name]; ok {
			raw[name] = tftypes.NewValue(typ, value)
		} else {
			raw[name] = tftypes.NewValue(typ, nil)
		}
	}
	return tfsdk.Plan{Schema: schema.Schema, Raw: tftypes.NewValue(objectType, raw)}
}

func TestPortalFormResourceLifecycle(t *testing.T) {
	const fields = `[{"type":"text","name":"api_id","label":"API"},{"name":"submit","type":"submit","value":"Register"}]`
	const changedFields = `[{"type":"text","name":"api_id","label":"API ID"},{"name":"submit","type":"submit","value":"Register"}]`
	const created = `{"id":"form-id","name":"api-registration","type":"api_registration","status":"unpublished","fields":[{"type":"text","name":"api_id","label":"API","required":false,"built_in":true},{"name":"submit","type":"submit","value":"Register","built_in":false}],"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z"}`
	const updated = `{"id":"form-id","name":"api-registration","type":"api_registration","status":"published","fields":[{"type":"text","name":"api_id","label":"API ID","built_in":true},{"name":"submit","type":"submit","value":"Register","built_in":false}],"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}`
	current := created
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Accept") != "application/json" {
			t.Errorf("missing authentication or accept header")
		}
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			if r.Header.Get("Content-Type") != "application/json" ||
				(r.Method == http.MethodPost && r.URL.Path != "/v3/portals/portal-id/forms") ||
				(r.Method == http.MethodPut && r.URL.Path != "/v3/portals/portal-id/forms/form-id") {
				t.Errorf("unexpected write request: %s %s", r.Method, r.URL)
			}
			body, _ := io.ReadAll(r.Body)
			var payload struct {
				Name   string            `json:"name"`
				Type   string            `json:"type"`
				Status string            `json:"status"`
				Fields []json.RawMessage `json:"fields"`
			}
			if err := json.Unmarshal(body, &payload); err != nil || len(payload.Fields) != 2 ||
				payload.Name != "api-registration" || payload.Type != "api_registration" {
				t.Errorf("invalid form payload: %s, error: %v", body, err)
			}
			var firstField struct {
				Label string `json:"label"`
			}
			if len(payload.Fields) > 0 {
				json.Unmarshal(payload.Fields[0], &firstField)
			}
			if r.Method == http.MethodPost {
				if payload.Status != "unpublished" || firstField.Label != "API" {
					t.Errorf("unexpected create payload: %s", body)
				}
				w.WriteHeader(http.StatusCreated)
			} else {
				if payload.Status != "published" || firstField.Label != "API ID" {
					t.Errorf("unexpected update payload: %s", body)
				}
				current = updated
				w.WriteHeader(http.StatusOK)
			}
			fmt.Fprint(w, current)
		case http.MethodGet:
			if r.URL.Path != "/v3/portals/portal-id/forms/form-id" {
				t.Errorf("unexpected read path: %s", r.URL)
			}
			if deleted {
				w.WriteHeader(http.StatusNotFound)
			} else {
				fmt.Fprint(w, current)
			}
		case http.MethodDelete:
			if r.URL.Path != "/v3/portals/portal-id/forms/form-id" {
				t.Errorf("unexpected delete path: %s", r.URL)
			}
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	r := &PortalFormResource{client: &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("invalid schema: %v", schemaResp.Diagnostics)
	}
	plan := formPlan(t, schemaResp, map[string]string{
		"portal_id": "portal-id", "name": "api-registration", "type": "api_registration",
		"status": "unpublished", "fields_json": fields,
	})
	create := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &create)
	if create.Diagnostics.HasError() {
		t.Fatalf("create failed: %v", create.Diagnostics)
	}
	var state PortalFormResourceModel
	if diags := create.State.Get(ctx, &state); diags.HasError() ||
		state.ID.ValueString() != "form-id" || state.FieldsJSON.ValueString() != fields {
		t.Fatalf("unexpected created state: %+v, %v", state, diags)
	}

	read := resource.ReadResponse{State: create.State}
	r.Read(ctx, resource.ReadRequest{State: create.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatalf("read failed: %v", read.Diagnostics)
	}
	if diags := read.State.Get(ctx, &state); diags.HasError() || state.FieldsJSON.ValueString() != fields {
		t.Fatalf("API defaults caused a diff: %+v, %v", state, diags)
	}

	updatePlan := formPlan(t, schemaResp, map[string]string{
		"portal_id": "portal-id", "name": "api-registration", "type": "api_registration",
		"status": "published", "fields_json": changedFields, "id": "form-id",
		"created_at": "2026-01-02T03:04:05Z", "updated_at": "2026-01-02T03:04:05Z",
	})
	update := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: read.State}, &update)
	if update.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", update.Diagnostics)
	}
	if diags := update.State.Get(ctx, &state); diags.HasError() ||
		state.Status.ValueString() != "published" || state.FieldsJSON.ValueString() != changedFields ||
		state.UpdatedAt.ValueString() != "2026-01-03T03:04:05Z" {
		t.Fatalf("unexpected updated state: %+v, %v", state, diags)
	}
	current = `{"id":"form-id","name":"api-registration","type":"api_registration","status":"unpublished","fields":[{"type":"text","name":"api_id","label":"Externally changed","built_in":true},{"name":"submit","type":"submit","value":"Register","built_in":false}],"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-04T03:04:05Z"}`
	drift := resource.ReadResponse{State: update.State}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &drift)
	if drift.Diagnostics.HasError() {
		t.Fatalf("read drift failed: %v", drift.Diagnostics)
	}
	if diags := drift.State.Get(ctx, &state); diags.HasError() ||
		state.Status.ValueString() != "unpublished" || state.FieldsJSON.ValueString() == changedFields {
		t.Fatalf("external changes not detected: %+v, %v", state, diags)
	}
	del := resource.DeleteResponse{State: drift.State}
	r.Delete(ctx, resource.DeleteRequest{State: drift.State}, &del)
	if del.Diagnostics.HasError() {
		t.Fatalf("delete failed: %v", del.Diagnostics)
	}
	missing := resource.ReadResponse{State: drift.State}
	r.Read(ctx, resource.ReadRequest{State: drift.State}, &missing)
	if missing.Diagnostics.HasError() || !missing.State.Raw.IsNull() {
		t.Fatalf("missing form not removed from state: %v", missing.Diagnostics)
	}
}

func TestPortalFormResourceCreateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"detail":"form already exists"}`)
	}))
	defer server.Close()
	ctx := context.Background()
	r := &PortalFormResource{client: &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	plan := formPlan(t, schemaResp, map[string]string{
		"portal_id": "portal-id", "name": "api-registration", "type": "api_registration",
		"status": "unpublished", "fields_json": `[{"name":"submit","type":"submit","value":"Register"}]`,
	})
	resp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
	if !resp.Diagnostics.HasError() || resp.State.Raw.IsKnown() && !resp.State.Raw.IsNull() {
		t.Fatalf("API error did not prevent resource creation: %v", resp.Diagnostics)
	}
}

func TestPortalFormResourceUpdateDeveloperRegistration(t *testing.T) {
	const fields = `[{"name":"full_name","type":"text","label":"Full Name"},{"name":"email","type":"email","label":"Email"},{"name":"submit","type":"submit","value":"Join"}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v3/portals/portal-id/forms/form-id" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		var payload map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("invalid update payload: %v", err)
		}
		if _, ok := payload["name"]; ok {
			t.Error("developer registration update must not send name")
		}
		fmt.Fprint(w, `{"id":"form-id","name":"developer-registration","type":"developer_registration","status":"published","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}`)
	}))
	defer server.Close()
	ctx := context.Background()
	r := &PortalFormResource{client: &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	plan := formPlan(t, schemaResp, map[string]string{
		"portal_id": "portal-id", "id": "form-id", "name": "developer-registration",
		"type": "developer_registration", "status": "published", "fields_json": fields,
	})
	prior := tfsdk.State{Schema: schemaResp.Schema}
	if diags := prior.Set(ctx, &PortalFormResourceModel{
		PortalID: types.StringValue("portal-id"), ID: types.StringValue("form-id"),
		Name: types.StringValue("developer-registration"), Type: types.StringValue("developer_registration"),
		Status: types.StringValue("unpublished"), FieldsJSON: types.StringValue(fields),
		CreatedAt: types.StringValue("2026-01-02T03:04:05Z"), UpdatedAt: types.StringValue("2026-01-02T03:04:05Z"),
	}); diags.HasError() {
		t.Fatalf("invalid prior state: %v", diags)
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: prior}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
}

func TestPortalFormResourceInvalidFields(t *testing.T) {
	for _, fields := range []string{"not JSON", `[]`, `{}`} {
		t.Run(fields, func(t *testing.T) {
			ctx := context.Background()
			r := &PortalFormResource{}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			plan := formPlan(t, schemaResp, map[string]string{
				"portal_id": "portal-id", "name": "api-registration", "type": "api_registration",
				"status": "unpublished", "fields_json": fields,
			})
			resp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("invalid field definitions accepted")
			}
		})
	}
}
