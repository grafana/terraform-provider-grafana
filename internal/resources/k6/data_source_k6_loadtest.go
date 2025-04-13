package k6

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*loadTestDataSource)(nil)
)

var (
	dataSourceLoadTestName = "grafana_k6_load_test"
)

func dataSourceLoadTest() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceLoadTestName,
		&loadTestDataSource{},
	)
}

// loadTestDataSourceModel maps the data source schema data.
type loadTestDataSourceModel struct {
	ID                types.Int32  `tfsdk:"id"`
	ProjectID         types.Int32  `tfsdk:"project_id"`
	Name              types.String `tfsdk:"name"`
	Script            types.String `tfsdk:"script"`
	BaselineTestRunId types.Int32  `tfsdk:"baseline_test_run_id"`
	Created           types.String `tfsdk:"created"`
	Updated           types.String `tfsdk:"updated"`
}

// loadTestDataSource is the data source implementation.
type loadTestDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *loadTestDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceLoadTestName
}

// Schema defines the schema for the data source.
func (d *loadTestDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a k6 load test.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Description: "Numeric identifier of the load test.",
				Required:    true,
			},
			"project_id": schema.Int32Attribute{
				Description: "The identifier of the project this load test belongs to.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the load test.",
				Computed:    true,
			},
			"script": schema.StringAttribute{
				Description: "The k6 test script content.",
				Computed:    true,
			},
			"baseline_test_run_id": schema.Int32Attribute{
				Description: "Identifier of a baseline test run used for results comparison.",
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: "The date when the load test was created.",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "The date when the load test was last updated.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *loadTestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state loadTestDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve the load test attributes
	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.LoadTestsAPI.LoadTestsRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(d.config.StackID)

	lt, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 load test",
			"Could not read k6 load test ID "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Retrieve the load test script content
	scriptReq := d.client.LoadTestsAPI.LoadTestsScriptRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(d.config.StackID)

	script, _, err := scriptReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 load test script",
			"Could not read k6 load test script with id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(lt.GetName())
	state.ProjectID = types.Int32Value(lt.GetProjectId())
	state.BaselineTestRunId = handleBaselineTestRunId(lt.GetBaselineTestRunId())
	state.Script = types.StringValue(script)
	state.Created = types.StringValue(lt.GetCreated().Format(time.RFC3339Nano))
	state.Updated = types.StringValue(lt.GetUpdated().Format(time.RFC3339Nano))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func handleBaselineTestRunId(baselineTestRunId int32) types.Int32 {
	if baselineTestRunId == 0 {
		// If the API returned 0, set it as null
		return types.Int32Null()
	}
	return types.Int32Value(baselineTestRunId)
}
