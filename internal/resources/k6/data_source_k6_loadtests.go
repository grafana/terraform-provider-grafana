package k6

import (
	"context"
	"slices"
	"sort"
	"strconv"
	"strings"
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
	_ datasource.DataSourceWithConfigure = (*loadTestsDataSource)(nil)
)

var (
	dataSourceLoadTestsName = "grafana_k6_load_tests"
)

func dataSourceLoadTests() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceLoadTestsName,
		&loadTestsDataSource{},
	)
}

// loadTestsDataSourceModel maps the data source schema data.
type loadTestsDataSourceModel struct {
	ID        types.Int32               `tfsdk:"id"`
	ProjectID types.Int32               `tfsdk:"project_id"`
	Name      types.String              `tfsdk:"name"`
	LoadTests []loadTestDataSourceModel `tfsdk:"load_tests"`
}

// loadTestsDataSource is the data source implementation.
type loadTestsDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *loadTestsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceLoadTestsName
}

// Schema defines the schema for the data source.
func (d *loadTestsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves all k6 load tests that belong to a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Description: "The identifier of the project the load tests belong to. This is the same as the project_id.",
				Computed:    true,
			},
			"project_id": schema.Int32Attribute{
				Description: "The identifier of the project the load tests belong to.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the load test.",
				Optional:    true,
			},
			"load_tests": schema.ListAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":                   types.Int32Type,
						"name":                 types.StringType,
						"project_id":           types.Int32Type,
						"script":               types.StringType,
						"baseline_test_run_id": types.Int32Type,
						"created":              types.StringType,
						"updated":              types.StringType,
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *loadTestsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state loadTestsDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the project_id
	state.ID = state.ProjectID

	// Retrieve the project's load tests
	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.LoadTestsAPI.ProjectsLoadTestsRetrieve(ctx, state.ProjectID.ValueInt32()).
		XStackId(d.config.StackID)

	lts, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 load tests",
			"Could not read k6 load tests with project id "+strconv.Itoa(int(state.ProjectID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Process the results and populate the state with the retrieved load tests
	var loadTestStates []loadTestDataSourceModel
	if !state.Name.IsNull() {
		lts.Value = slices.DeleteFunc(lts.Value, func(lt k6.LoadTestApiModel) bool {
			return !strings.EqualFold(lt.GetName(), state.Name.ValueString())
		})
	}
	sort.Slice(lts.Value, func(i, j int) bool {
		return lts.Value[i].GetCreated().Before(lts.Value[j].GetCreated())
	})
	for _, lt := range lts.Value {
		// Retrieve the load test script content
		scriptReq := d.client.LoadTestsAPI.LoadTestsScriptRetrieve(ctx, lt.GetId()).
			XStackId(d.config.StackID)

		script, _, err := scriptReq.Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading k6 load test script",
				"Could not read k6 load test script with id"+strconv.Itoa(int(lt.GetId()))+": "+err.Error(),
			)
			return
		}

		// For each load test, populate the state
		ltState := loadTestDataSourceModel{
			ID:                types.Int32Value(lt.GetId()),
			Name:              types.StringValue(lt.GetName()),
			ProjectID:         types.Int32Value(lt.GetProjectId()),
			BaselineTestRunID: handleBaselineTestRunID(lt.GetBaselineTestRunId()),
			Script:            types.StringValue(script),
			Created:           types.StringValue(lt.GetCreated().Format(time.RFC3339Nano)),
			Updated:           types.StringValue(lt.GetUpdated().Format(time.RFC3339Nano)),
		}

		// Add the load test state to the list
		loadTestStates = append(loadTestStates, ltState)
	}

	state.LoadTests = loadTestStates

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
