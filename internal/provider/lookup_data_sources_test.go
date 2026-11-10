package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func lookupRequest(t *testing.T, d datasource.DataSource, inputs map[string]string) (datasource.ReadRequest, datasource.ReadResponse) {
	t.Helper()
	ctx := context.Background()
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("invalid schema: %v", schemaResp.Diagnostics)
	}
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, typ := range objectType.AttributeTypes {
		if value, ok := inputs[name]; ok {
			values[name] = tftypes.NewValue(typ, value)
		} else {
			values[name] = tftypes.NewValue(typ, nil)
		}
	}
	return datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, values)},
	}, datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
}

func TestApplicationAuthStrategyDataSourceRead(t *testing.T) {
	const success = `{"data":[{"id":"strategy-id","name":"API & Key","display_name":"API Key","strategy_type":"key_auth","active":true,"dcr_provider":null,"supports_multiple_credentials":true,"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}]}`
	tests := []struct {
		name         string
		status       int
		body         string
		wantError    bool
		wantDCR      string
		wantType     string
		wantMultiple bool
		wantActive   bool
	}{
		{name: "key auth", status: http.StatusOK, body: success, wantType: "key_auth", wantMultiple: true, wantActive: true},
		{name: "OIDC with DCR", status: http.StatusOK, body: `{"data":[{"id":"strategy-id","name":"API & Key","strategy_type":"openid_connect","active":false,"dcr_provider":{"id":"dcr-id"},"supports_multiple_credentials":false,"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}]}`, wantType: "openid_connect", wantDCR: "dcr-id"},
		{name: "not found", status: http.StatusOK, body: `{"data":[]}`, wantError: true},
		{name: "wrong name", status: http.StatusOK, body: `{"data":[{"name":"other"}]}`, wantError: true},
		{name: "duplicate", status: http.StatusOK, body: `{"data":[{"name":"API & Key"},{"name":"API & Key"}]}`, wantError: true},
		{name: "API error", status: http.StatusForbidden, body: `{"detail":"forbidden"}`, wantError: true},
		{name: "invalid JSON", status: http.StatusOK, body: `{`, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v2/application-auth-strategies" ||
					r.URL.Query().Get("filter[name][eq]") != "API & Key" ||
					r.Header.Get("Authorization") != "Bearer test-token" ||
					r.Header.Get("Accept") != "application/json" {
					t.Errorf("unexpected request: %s %s, headers %v", r.Method, r.URL, r.Header)
				}
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			d := &ApplicationAuthStrategyDataSource{client: &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}}
			req, resp := lookupRequest(t, d, map[string]string{"name": "API & Key"})
			d.Read(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if tt.wantError {
				return
			}
			var state ApplicationAuthStrategyDataSourceModel
			if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("invalid state: %v", diags)
			}
			if state.ID.ValueString() != "strategy-id" || state.Name.ValueString() != "API & Key" ||
				state.StrategyType.ValueString() != tt.wantType ||
				state.Active.ValueBool() != tt.wantActive ||
				state.SupportsMultipleCredentials.ValueBool() != tt.wantMultiple ||
				state.CreatedAt.ValueString() != "2026-01-02T03:04:05Z" {
				t.Fatalf("unexpected strategy state: %+v", state)
			}
			if tt.wantDCR == "" && !state.DCRProviderID.IsNull() ||
				tt.wantDCR != "" && state.DCRProviderID.ValueString() != tt.wantDCR {
				t.Fatalf("unexpected DCR provider ID: %v", state.DCRProviderID)
			}
		})
	}
}

func TestLookupDataSourcesRead(t *testing.T) {
	const success = `{"data":[{"id":"form-id","name":"api-registration","type":"api_registration","status":"published","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}]}`
	tests := []struct {
		name      string
		status    int
		body      string
		wantError bool
	}{
		{"found", http.StatusOK, success, false},
		{"not found", http.StatusOK, `{"data":[]}`, true},
		{"wrong name", http.StatusOK, `{"data":[{"name":"other"}]}`, true},
		{"duplicate", http.StatusOK, `{"data":[{"name":"api-registration"},{"name":"api-registration"}]}`, true},
		{"API error", http.StatusNotFound, `{"detail":"portal not found"}`, true},
		{"invalid JSON", http.StatusOK, `{`, true},
	}

	runNewLookupDataSourcesRead := func(t *testing.T) {
		tests := []struct {
			name       string
			dataSource datasource.DataSource
			inputs     map[string]string
			path       string
			query      string
			body       string
			wantError  bool
		}{
			{
				name:       "portal team",
				dataSource: &PortalTeamDataSource{},
				inputs:     map[string]string{"portal_id": "portal-id", "name": "Platform"},
				path:       "/v3/portals/portal-id/teams",
				query:      "filter[name][eq]",
				body:       `{"data":[{"id":"team-id","name":"Platform","description":"Platform team","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z","can_own_applications":true,"konnect_managed":false}]}`,
			},
			{
				name:       "portal developer",
				dataSource: &PortalDeveloperDataSource{},
				inputs:     map[string]string{"portal_id": "portal-id", "name": "Jane Doe"},
				path:       "/v3/portals/portal-id/developers",
				query:      "filter[name][eq]",
				body:       `{"data":[{"id":"developer-id","email":"jane@example.com","full_name":"Jane Doe","status":"approved","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}]}`,
			},
			{
				name:       "portal application",
				dataSource: &PortalApplicationDataSource{},
				inputs:     map[string]string{"portal_id": "portal-id", "name": "My App"},
				path:       "/v3/portals/portal-id/applications",
				query:      "filter[name][eq]",
				body:       `{"data":[{"id":"application-id","name":"My App","description":null,"client_id":null,"registration_count":2,"auth_strategy":{"id":"strategy-id"},"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}]}`,
			},
			{
				name:       "catalog APIs",
				dataSource: &APIsDataSource{},
				inputs:     map[string]string{"name": "Payments"},
				path:       "/v3/apis",
				query:      "filter[name][eq]",
				body:       `{"data":[{"id":"api-v1","name":"Payments","version":"v1","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"},{"id":"api-v2","name":"Payments","version":"v2","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}],"meta":{"page":{"number":1,"size":100,"total":2}}}`,
			},
			{
				name:       "catalog API by name",
				dataSource: &APIByNameDataSource{},
				inputs:     map[string]string{"name": "Payments"},
				path:       "/v3/apis",
				query:      "filter[name][eq]",
				body:       `{"data":[{"id":"api-id","name":"Payments","version":"v1","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}],"meta":{"page":{"number":1,"size":100,"total":1}}}`,
			},
			{
				name:       "catalog API by name and version",
				dataSource: &APIDataSource{},
				inputs:     map[string]string{"name": "Payments", "version": "v1"},
				path:       "/v3/apis",
				query:      "filter[version][eq]",
				body:       `{"data":[{"id":"api-id","name":"Payments","version":"v1","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T03:04:05Z"}],"meta":{"page":{"number":1,"size":100,"total":1}}}`,
			},
			{
				name:       "ambiguous catalog API by name",
				dataSource: &APIByNameDataSource{},
				inputs:     map[string]string{"name": "Payments"},
				path:       "/v3/apis",
				query:      "filter[name][eq]",
				body:       `{"data":[{"id":"api-v1","name":"Payments"},{"id":"api-v2","name":"Payments"}],"meta":{"page":{"number":1,"size":100,"total":2}}}`,
				wantError:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedQueryValue := tt.inputs["name"]
					if tt.query == "filter[version][eq]" {
						expectedQueryValue = tt.inputs["version"]
					}
					if r.Method != http.MethodGet || r.URL.Path != tt.path ||
						r.URL.Query().Get(tt.query) != expectedQueryValue ||
						r.Header.Get("Authorization") != "Bearer test-token" ||
						r.Header.Get("Accept") != "application/json" {
						t.Errorf("unexpected request: %s %s, headers %v", r.Method, r.URL, r.Header)
					}
					if r.URL.Path == "/v3/apis" && r.URL.Query().Get("page[size]") != "100" {
						t.Errorf("expected API page size of 100, got %q", r.URL.Query().Get("page[size]"))
					}
					fmt.Fprint(w, tt.body)
				}))
				defer server.Close()

				switch d := tt.dataSource.(type) {
				case *PortalTeamDataSource:
					d.client = &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}
				case *PortalDeveloperDataSource:
					d.client = &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}
				case *PortalApplicationDataSource:
					d.client = &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}
				case *APIsDataSource:
					d.client = &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}
				case *APIByNameDataSource:
					d.client = &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}
				case *APIDataSource:
					d.client = &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}
				}

				req, resp := lookupRequest(t, tt.dataSource, tt.inputs)
				tt.dataSource.Read(context.Background(), req, &resp)
				if resp.Diagnostics.HasError() != tt.wantError {
					t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
				}
			})
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v3/portals/portal-id/forms" ||
					r.URL.Query().Get("filter[name][eq]") != "api-registration" ||
					r.Header.Get("Authorization") != "Bearer test-token" ||
					r.Header.Get("Accept") != "application/json" {
					t.Errorf("unexpected request: %s %s, headers %v", r.Method, r.URL, r.Header)
				}
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			d := &PortalFormDataSource{client: &KonnectExtraClient{HTTPClient: server.Client(), ServerURL: server.URL, Token: "test-token"}}
			req, resp := lookupRequest(t, d, map[string]string{"portal_id": "portal-id", "name": "api-registration"})
			d.Read(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if tt.wantError {
				return
			}
			var state PortalFormDataSourceModel
			if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("invalid state: %v", diags)
			}
			if state.ID != types.StringValue("form-id") || state.Name != types.StringValue("api-registration") ||
				state.Type != types.StringValue("api_registration") || state.Status != types.StringValue("published") ||
				state.UpdatedAt != types.StringValue("2026-01-03T03:04:05Z") {
				t.Fatalf("unexpected form state: %+v", state)
			}
		})
	}
	runNewLookupDataSourcesRead(t)
}
