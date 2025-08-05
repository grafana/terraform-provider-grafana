package k6

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*projectAllowedLoadZonesDataSource)(nil)
)

var (
	dataSourceProjectAllowedLoadZonesName = "grafana_k6_project_allowed_load_zones"
)

func dataSourceProjectAllowedLoadZones() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceProjectAllowedLoadZonesName,
		&projectAllowedLoadZonesDataSource{},
	)
}

// projectAllowedLoadZonesDataSourceModel maps the data source schema data.
type projectAllowedLoadZonesDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AllowedLoadZones types.List   `tfsdk:"allowed_load_zones"`
}

// projectAllowedLoadZonesDataSource is the data source implementation.
type projectAllowedLoadZonesDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *projectAllowedLoadZonesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceProjectAllowedLoadZonesName
}

// Schema defines the schema for the data source.
func (d *projectAllowedLoadZonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves allowed load zones for a k6 project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the project allowed load zones. This is set to the same as the project_id.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The identifier of the project to retrieve allowed load zones for.",
				Required:    true,
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
func (d *projectAllowedLoadZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectAllowedLoadZonesDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	intID, err := strconv.ParseInt(state.ProjectID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project ID",
			"Could not parse project ID '"+state.ProjectID.ValueString()+"': "+err.Error(),
		)
		return
	}
	projectID := int32(intID)

	// Get allowed load zones
	allowedZones, err := getProjectAllowedLoadZones(ctx, d.client, d.config, projectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading allowed load zones",
			"Could not read allowed load zones for k6 project: "+err.Error(),
		)
		return
	}

	// Set ID to match project_id
	state.ID = state.ProjectID

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
