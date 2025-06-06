package k6

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*projectLimitsDataSource)(nil)
)

var (
	dataSourceProjectLimitsName = "grafana_k6_project_limits"
)

func dataSourceProjectLimits() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceProjectLimitsName,
		&projectLimitsDataSource{},
	)
}

// projectLimitsDataSourceModel maps the data source schema data.
type projectLimitsDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	VuhMaxPerMonth      types.Int32  `tfsdk:"vuh_max_per_month"`
	VuMaxPerTest        types.Int32  `tfsdk:"vu_max_per_test"`
	VuBrowserMaxPerTest types.Int32  `tfsdk:"vu_browser_max_per_test"`
	DurationMaxPerTest  types.Int32  `tfsdk:"duration_max_per_test"`
}

// projectLimitsDataSource is the data source implementation.
type projectLimitsDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *projectLimitsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceProjectLimitsName
}

// Schema defines the schema for the data source.
func (d *projectLimitsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a k6 project limits.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the project limits. This is set to the same as the project_id.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The identifier of the project to get limits for.",
				Required:    true,
			},
			"vuh_max_per_month": schema.Int32Attribute{
				Description: "Maximum amount of virtual user hours (VU/h) used per one calendar month.",
				Computed:    true,
			},
			"vu_max_per_test": schema.Int32Attribute{
				Description: "Maximum number of concurrent virtual users (VUs) used in one test.",
				Computed:    true,
			},
			"vu_browser_max_per_test": schema.Int32Attribute{
				Description: "Maximum number of concurrent browser virtual users (VUs) used in one test.",
				Computed:    true,
			},
			"duration_max_per_test": schema.Int32Attribute{
				Description: "Maximum duration of a test in seconds.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *projectLimitsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectLimitsDataSourceModel
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

	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.ProjectsAPI.ProjectsLimitsRetrieve(ctx, projectID).
		XStackId(d.config.StackID)

	limits, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 project limits",
			"Could not read k6 project limits with project id "+state.ProjectID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(strconv.Itoa(int(limits.GetProjectId())))
	state.ProjectID = types.StringValue(strconv.Itoa(int(limits.GetProjectId())))
	state.VuhMaxPerMonth = types.Int32Value(limits.GetVuhMaxPerMonth())
	state.VuMaxPerTest = types.Int32Value(limits.GetVuMaxPerTest())
	state.VuBrowserMaxPerTest = types.Int32Value(limits.GetVuBrowserMaxPerTest())
	state.DurationMaxPerTest = types.Int32Value(limits.GetDurationMaxPerTest())

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
