package k6

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*projectDataSource)(nil)
)

var (
	dataSourceProjectName = "grafana_k6_project"
)

func dataSourceProject() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceProjectName,
		&projectDataSource{},
	)
}

// projectDataSourceModel maps the data source schema data.
type projectDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	IsDefault        types.Bool   `tfsdk:"is_default"`
	GrafanaFolderUID types.String `tfsdk:"grafana_folder_uid"`
	Created          types.String `tfsdk:"created"`
	Updated          types.String `tfsdk:"updated"`
	AllowedLoadZones types.List   `tfsdk:"allowed_load_zones"`
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *projectDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceProjectName
}

// Schema defines the schema for the data source.
func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a k6 project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the project.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the project.",
				Computed:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Whether this project is the default for running tests when no explicit project identifier is provided.",
				Computed:    true,
			},
			"grafana_folder_uid": schema.StringAttribute{
				Description: "The Grafana folder uid.",
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: "The date when the project was created.",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "The date when the project was last updated.",
				Computed:    true,
			},
			"allowed_load_zones": schema.ListAttribute{
				Description: "List of allowed k6 load zone IDs for this project.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	intID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project ID",
			"Could not parse project ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	projectID := int32(intID)

	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.ProjectsAPI.ProjectsRetrieve(ctx, projectID).
		XStackId(d.config.StackID)

	p, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 project",
			"Could not read k6 project with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(p.GetName())
	state.IsDefault = types.BoolValue(p.GetIsDefault())
	state.GrafanaFolderUID = handleGrafanaFolderUID(p.GrafanaFolderUid)
	state.Created = types.StringValue(p.GetCreated().Format(time.RFC3339Nano))
	state.Updated = types.StringValue(p.GetUpdated().Format(time.RFC3339Nano))

	// Get allowed load zones
	allowedZones, err := d.getAllowedLoadZones(ctx, p.GetId())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading allowed load zones",
			"Could not read allowed load zones for k6 project: "+err.Error(),
		)
		return
	}

	// Convert to types.List
	var zoneValues []attr.Value
	for _, zone := range allowedZones {
		zoneValues = append(zoneValues, types.StringValue(zone))
	}
	state.AllowedLoadZones, diags = types.ListValue(types.StringType, zoneValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func handleGrafanaFolderUID(grafanaFolderUID k6.NullableString) types.String {
	if !grafanaFolderUID.IsSet() {
		return types.StringNull()
	}
	return types.StringValue(*grafanaFolderUID.Get())
}

// getAllowedLoadZones retrieves the allowed load zones for a project
// Returns k6_load_zone_ids directly from the API response
func (d *projectDataSource) getAllowedLoadZones(ctx context.Context, projectID int32) ([]string, error) {
	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	resp, _, err := d.client.LoadZonesAPI.ProjectsAllowedLoadZonesRetrieve(ctx, projectID).
		XStackId(d.config.StackID).
		Execute()
	if err != nil {
		return nil, err
	}

	var k6LoadZoneIds []string
	for _, zone := range resp.GetValue() {
		k6LoadZoneIds = append(k6LoadZoneIds, zone.GetK6LoadZoneId())
	}

	return k6LoadZoneIds, nil
}
