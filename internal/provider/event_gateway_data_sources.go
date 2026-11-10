package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type eventPage struct {
	Data json.RawMessage `json:"data"`
	Meta struct {
		Page struct {
			After string `json:"after"`
		} `json:"page"`
	} `json:"meta"`
}

func listEventItems[T any](ctx context.Context, d *lookupDataSource, path, name string) ([]T, error) {
	query := url.Values{"filter[name][contains]": []string{name}, "page[size]": []string{"100"}}
	var items []T
	for {
		var page eventPage
		if err := d.get(ctx, path, query, &page); err != nil {
			return nil, err
		}
		var pageItems []T
		if err := json.Unmarshal(page.Data, &pageItems); err != nil {
			return nil, fmt.Errorf("parsing collection data: %w", err)
		}
		items = append(items, pageItems...)
		if page.Meta.Page.After == "" {
			return items, nil
		}
		query.Set("page[after]", page.Meta.Page.After)
	}
}

func eventLabels(labels map[string]*string) map[string]types.String {
	result := make(map[string]types.String, len(labels))
	for key, value := range labels {
		result[key] = types.StringPointerValue(value)
	}
	return result
}

type eventGatewayItem struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Description          *string            `json:"description"`
	Labels               map[string]*string `json:"labels"`
	MinRuntimeVersion    string             `json:"min_runtime_version"`
	NodesTotal           int64              `json:"nodes_total"`
	VirtualClustersTotal int64              `json:"virtual_clusters_total"`
	Version              string             `json:"version"`
}

type EventGatewayDataSource struct{ lookupDataSource }

type EventGatewayDataSourceModel struct {
	Name                 types.String            `tfsdk:"name"`
	ID                   types.String            `tfsdk:"id"`
	Description          types.String            `tfsdk:"description"`
	Labels               map[string]types.String `tfsdk:"labels"`
	MinRuntimeVersion    types.String            `tfsdk:"min_runtime_version"`
	NodesTotal           types.Int64             `tfsdk:"nodes_total"`
	VirtualClustersTotal types.Int64             `tfsdk:"virtual_clusters_total"`
	Version              types.String            `tfsdk:"version"`
}

func NewEventGatewayDataSource() datasource.DataSource { return &EventGatewayDataSource{} }
func (d *EventGatewayDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_gateway"
}
func (d *EventGatewayDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Look up an Event Gateway by name.", Attributes: map[string]schema.Attribute{
		"name": requiredStringAttribute("The name of the Event Gateway to look up."),
		"id":   computedStringAttribute("The ID of the Event Gateway."), "description": computedStringAttribute("The Event Gateway description."),
		"labels":                 schema.MapAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "The Event Gateway labels."},
		"min_runtime_version":    computedStringAttribute("The minimum Event Gateway runtime version."),
		"nodes_total":            schema.Int64Attribute{Computed: true, MarkdownDescription: "The total number of nodes."},
		"virtual_clusters_total": schema.Int64Attribute{Computed: true, MarkdownDescription: "The total number of virtual clusters."},
		"version":                computedStringAttribute("The Event Gateway version."),
	}}
}
func (d *EventGatewayDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}
func (d *EventGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EventGatewayDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := listEventItems[eventGatewayItem](ctx, &d.lookupDataSource, "/v1/event-gateways", data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error looking up Event Gateway", err.Error())
		return
	}
	var match *eventGatewayItem
	for i := range items {
		if items[i].Name == data.Name.ValueString() {
			if match != nil {
				resp.Diagnostics.AddError("Event Gateway is ambiguous", fmt.Sprintf("More than one Event Gateway named %q was found.", data.Name.ValueString()))
				return
			}
			match = &items[i]
		}
	}
	if match == nil {
		resp.Diagnostics.AddError("Event Gateway not found", fmt.Sprintf("No Event Gateway named %q was found.", data.Name.ValueString()))
		return
	}
	data.ID, data.Description, data.Labels = types.StringValue(match.ID), types.StringPointerValue(match.Description), eventLabels(match.Labels)
	data.MinRuntimeVersion, data.NodesTotal, data.VirtualClustersTotal, data.Version = types.StringValue(match.MinRuntimeVersion), types.Int64Value(match.NodesTotal), types.Int64Value(match.VirtualClustersTotal), types.StringValue(match.Version)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type eventGatewayChildItem struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Labels      map[string]*string `json:"labels"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type EventGatewayBackendClusterDataSource struct{ lookupDataSource }
type EventGatewayBackendClusterDataSourceModel struct {
	GatewayID   types.String            `tfsdk:"gateway_id"`
	Name        types.String            `tfsdk:"name"`
	ID          types.String            `tfsdk:"id"`
	Description types.String            `tfsdk:"description"`
	Labels      map[string]types.String `tfsdk:"labels"`
	CreatedAt   types.String            `tfsdk:"created_at"`
	UpdatedAt   types.String            `tfsdk:"updated_at"`
}

func NewEventGatewayBackendClusterDataSource() datasource.DataSource {
	return &EventGatewayBackendClusterDataSource{}
}
func (d *EventGatewayBackendClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_gateway_backend_cluster"
}
func eventGatewayChildSchema(description string) schema.Schema {
	return schema.Schema{MarkdownDescription: description, Attributes: map[string]schema.Attribute{
		"gateway_id": requiredStringAttribute("The ID of the Event Gateway."), "name": requiredStringAttribute("The name to look up."),
		"id": computedStringAttribute("The ID."), "description": computedStringAttribute("The description."),
		"labels":     schema.MapAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Labels."},
		"created_at": computedStringAttribute("An ISO-8601 creation timestamp."), "updated_at": computedStringAttribute("An ISO-8601 update timestamp."),
	}}
}
func (d *EventGatewayBackendClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = eventGatewayChildSchema("Look up an Event Gateway backend cluster by name.")
}
func (d *EventGatewayBackendClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}
func setEventGatewayChild(ctx context.Context, d *lookupDataSource, gatewayID, name, suffix, title string, set func(eventGatewayChildItem)) error {
	items, err := listEventItems[eventGatewayChildItem](ctx, d, fmt.Sprintf("/v1/event-gateways/%s/%s", url.PathEscape(gatewayID), suffix), name)
	if err != nil {
		return err
	}
	var match *eventGatewayChildItem
	for i := range items {
		if items[i].Name == name {
			if match != nil {
				return fmt.Errorf("more than one %s named %q was found", title, name)
			}
			match = &items[i]
		}
	}
	if match == nil {
		return fmt.Errorf("no %s named %q was found", title, name)
	}
	set(*match)
	return nil
}
func (d *EventGatewayBackendClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EventGatewayBackendClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := setEventGatewayChild(ctx, &d.lookupDataSource, data.GatewayID.ValueString(), data.Name.ValueString(), "backend-clusters", "backend cluster", func(item eventGatewayChildItem) {
		data.ID, data.Description, data.Labels = types.StringValue(item.ID), types.StringPointerValue(item.Description), eventLabels(item.Labels)
		data.CreatedAt, data.UpdatedAt = types.StringValue(item.CreatedAt.Format(time.RFC3339)), types.StringValue(item.UpdatedAt.Format(time.RFC3339))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error looking up Event Gateway backend cluster", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type EventGatewayListenerDataSource struct{ lookupDataSource }
type EventGatewayListenerDataSourceModel EventGatewayBackendClusterDataSourceModel

func NewEventGatewayListenerDataSource() datasource.DataSource {
	return &EventGatewayListenerDataSource{}
}
func (d *EventGatewayListenerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_gateway_listener"
}
func (d *EventGatewayListenerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = eventGatewayChildSchema("Look up an Event Gateway listener by name.")
}
func (d *EventGatewayListenerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}
func (d *EventGatewayListenerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EventGatewayListenerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := setEventGatewayChild(ctx, &d.lookupDataSource, data.GatewayID.ValueString(), data.Name.ValueString(), "listeners", "listener", func(item eventGatewayChildItem) {
		data.ID, data.Description, data.Labels = types.StringValue(item.ID), types.StringPointerValue(item.Description), eventLabels(item.Labels)
		data.CreatedAt, data.UpdatedAt = types.StringValue(item.CreatedAt.Format(time.RFC3339)), types.StringValue(item.UpdatedAt.Format(time.RFC3339))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error looking up Event Gateway listener", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type EventGatewaySchemaRegistryDataSource struct{ lookupDataSource }
type EventGatewaySchemaRegistryDataSourceModel EventGatewayBackendClusterDataSourceModel

func NewEventGatewaySchemaRegistryDataSource() datasource.DataSource {
	return &EventGatewaySchemaRegistryDataSource{}
}
func (d *EventGatewaySchemaRegistryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_gateway_schema_registry"
}
func (d *EventGatewaySchemaRegistryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = eventGatewayChildSchema("Look up an Event Gateway schema registry by name.")
}
func (d *EventGatewaySchemaRegistryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req, resp)
}
func (d *EventGatewaySchemaRegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EventGatewaySchemaRegistryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := setEventGatewayChild(ctx, &d.lookupDataSource, data.GatewayID.ValueString(), data.Name.ValueString(), "schema-registries", "schema registry", func(item eventGatewayChildItem) {
		data.ID, data.Description, data.Labels = types.StringValue(item.ID), types.StringPointerValue(item.Description), eventLabels(item.Labels)
		data.CreatedAt, data.UpdatedAt = types.StringValue(item.CreatedAt.Format(time.RFC3339)), types.StringValue(item.UpdatedAt.Format(time.RFC3339))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error looking up Event Gateway schema registry", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
