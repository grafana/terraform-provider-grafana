package k6

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/k6providerapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithConfigure   = (*loadTestResource)(nil)
	_ resource.ResourceWithImportState = (*loadTestResource)(nil)
)

var (
	resourceLoadTestName = "grafana_k6_load_test"
	resourceLoadTestID   = common.NewResourceID(common.IntIDField("id"))
)

func resourceLoadTest() *common.Resource {
	return common.NewResource(
		common.CategoryK6,
		resourceLoadTestName,
		resourceLoadTestID,
		&loadTestResource{},
	).WithLister(k6ListerFunction(listLoadTests))
}

// loadTestResourceModel maps the resource schema data.
type loadTestResourceModel struct {
	ID                types.Int32  `tfsdk:"id"`
	ProjectID         types.Int32  `tfsdk:"project_id"`
	Name              types.String `tfsdk:"name"`
	Script            types.String `tfsdk:"script"`
	BaselineTestRunID types.Int32  `tfsdk:"baseline_test_run_id"`
	Created           types.String `tfsdk:"created"`
	Updated           types.String `tfsdk:"updated"`
}

// loadTestResource is the resource implementation.
type loadTestResource struct {
	basePluginFrameworkResource
}

// Metadata returns the resource type name.
func (r *loadTestResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceLoadTestName
}

// Schema defines the schema for the resource.
func (r *loadTestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a k6 load test.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Description: "Numeric identifier of the load test.",
				Computed:    true,
			},
			"project_id": schema.Int32Attribute{
				Description: "The identifier of the project this load test belongs to.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the load test.",
				Required:    true,
			},
			"script": schema.StringAttribute{
				Description: "The k6 test script content. Can be provided inline or via the `file()` function.",
				Required:    true,
			},
			"baseline_test_run_id": schema.Int32Attribute{
				Description: "Identifier of a baseline test run used for results comparison.",
				Optional:    true,
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

// Create creates the resource and sets the Terraform state on success.
func (r *loadTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan loadTestResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.LoadTestsAPI.ProjectsLoadTestsCreate(ctx, plan.ProjectID.ValueInt32()).
		Name(plan.Name.ValueString()).
		Script(io.NopCloser(strings.NewReader(plan.Script.ValueString()))).
		XStackId(r.config.StackID)

	// Create new load test
	lt, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating load test",
			"Could not create load test, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.Int32Value(lt.GetId())
	plan.Name = types.StringValue(lt.GetName())
	plan.ProjectID = types.Int32Value(lt.GetProjectId())

	// Handle baseline_test_run_id properly
	if lt.GetBaselineTestRunId() == 0 && plan.BaselineTestRunID.IsNull() {
		// If the API returned 0 and the plan had it as null, keep it as null
		plan.BaselineTestRunID = types.Int32Null()
	} else {
		plan.BaselineTestRunID = types.Int32Value(lt.GetBaselineTestRunId())
	}

	plan.Created = types.StringValue(lt.GetCreated().Format(time.RFC3339Nano))
	plan.Updated = types.StringValue(lt.GetUpdated().Format(time.RFC3339Nano))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read retrieves the resource information.
func (r *loadTestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state loadTestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve the load test attributes
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.LoadTestsAPI.LoadTestsRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	lt, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading GCk6 load test",
			"Could not read GCk6 load test id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Retrieve the load test script content
	scriptReq := r.client.LoadTestsAPI.LoadTestsScriptRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	script, _, err := scriptReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading GCk6 load test script",
			"Could not read GCk6 load test script id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.ID = types.Int32Value(lt.GetId())
	state.Name = types.StringValue(lt.GetName())
	state.ProjectID = types.Int32Value(lt.GetProjectId())

	// Handle baseline_test_run_id properly
	if lt.GetBaselineTestRunId() == 0 {
		// If the API returned 0, set it as null
		state.BaselineTestRunID = types.Int32Null()
	} else {
		state.BaselineTestRunID = types.Int32Value(lt.GetBaselineTestRunId())
	}

	state.Script = types.StringValue(script)
	state.Created = types.StringValue(lt.GetCreated().Format(time.RFC3339Nano))
	state.Updated = types.StringValue(lt.GetUpdated().Format(time.RFC3339Nano))

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *loadTestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan loadTestResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state to retrieve the ID
	var state loadTestResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	toUpdate := k6.NewPatchLoadTestApiModel()
	toUpdate.SetName(plan.Name.ValueString())
	if plan.BaselineTestRunID.IsNull() {
		toUpdate.SetBaselineTestRunIdNil()
	} else {
		toUpdate.SetBaselineTestRunId(plan.BaselineTestRunID.ValueInt32())
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	updateReq := r.client.LoadTestsAPI.LoadTestsPartialUpdate(ctx, state.ID.ValueInt32()).
		PatchLoadTestApiModel(toUpdate).
		XStackId(r.config.StackID)

	// Update the load test
	_, err := updateReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating GCk6 load test",
			"Could not update GCk6 load test, unexpected error: "+err.Error(),
		)
		return
	}

	// Update the script if it has changed
	if !plan.Script.Equal(state.Script) {
		scriptReq := r.client.LoadTestsAPI.LoadTestsScriptUpdate(ctx, state.ID.ValueInt32()).
			Body(io.NopCloser(strings.NewReader(plan.Script.ValueString()))).
			XStackId(r.config.StackID)

		_, err = scriptReq.Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating GCk6 load test script",
				"Could not update GCk6 load test script, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Update resource state with updated items and timestamp
	fetchReq := r.client.LoadTestsAPI.LoadTestsRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	lt, _, err := fetchReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading GCk6 load test",
			"Could not read GCk6 load test id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Retrieve the updated script content
	scriptReq := r.client.LoadTestsAPI.LoadTestsScriptRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	script, _, err := scriptReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading GCk6 load test script",
			"Could not read GCk6 load test script id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	plan.ID = types.Int32Value(lt.GetId())
	plan.Name = types.StringValue(lt.GetName())
	plan.ProjectID = types.Int32Value(lt.GetProjectId())

	// Handle baseline_test_run_id properly
	if lt.GetBaselineTestRunId() == 0 {
		// If the API returned 0, set it as null
		plan.BaselineTestRunID = types.Int32Null()
	} else {
		plan.BaselineTestRunID = types.Int32Value(lt.GetBaselineTestRunId())
	}

	plan.Script = types.StringValue(script)
	plan.Created = types.StringValue(lt.GetCreated().Format(time.RFC3339Nano))
	plan.Updated = types.StringValue(lt.GetUpdated().Format(time.RFC3339Nano))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *loadTestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state loadTestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing load test
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	deleteReq := r.client.LoadTestsAPI.LoadTestsDestroy(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	_, err := deleteReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting GCk6 load test",
			"Could not delete GCk6 load test id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
	}
}

func (r *loadTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing GCk6 load test",
			"Could not parse GCk6 load test id "+req.ID+": "+err.Error(),
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("id"), types.Int32Value(int32(id)))

	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}

	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
}

// listLoadTests retrieves the list ids of all the existing load tests.
func listLoadTests(ctx context.Context, client *k6.APIClient, config *k6providerapi.K6APIConfig) ([]string, error) {
	ctx = context.WithValue(ctx, k6.ContextAccessToken, config.Token)
	resp, _, err := client.LoadTestsAPI.LoadTestsList(ctx).
		XStackId(config.StackID).
		Execute()
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, lt := range resp.Value {
		ids = append(ids, strconv.Itoa(int(lt.GetId())))
	}
	return ids, nil
}
