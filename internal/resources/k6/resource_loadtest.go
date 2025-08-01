package k6

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/k6providerapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithConfigure    = (*loadTestResource)(nil)
	_ resource.ResourceWithImportState  = (*loadTestResource)(nil)
	_ resource.ResourceWithUpgradeState = (*loadTestResource)(nil)
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
type loadTestResourceModelV0 struct {
	ID                types.Int32  `tfsdk:"id"`
	ProjectID         types.Int32  `tfsdk:"project_id"`
	Name              types.String `tfsdk:"name"`
	Script            types.String `tfsdk:"script"`
	BaselineTestRunID types.Int32  `tfsdk:"baseline_test_run_id"`
	Created           types.String `tfsdk:"created"`
	Updated           types.String `tfsdk:"updated"`
}

type loadTestResourceModelV1 struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	Name              types.String `tfsdk:"name"`
	Script            types.String `tfsdk:"script"`
	BaselineTestRunID types.String `tfsdk:"baseline_test_run_id"`
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
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the load test.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
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
			"baseline_test_run_id": schema.StringAttribute{
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
		Version: 1,
	}
}

func (r *loadTestResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
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
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// Convert int32 ID to string ID
				var priorStateData loadTestResourceModelV0
				diags := req.State.Get(ctx, &priorStateData)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := loadTestResourceModelV1{
					ID:                types.StringValue(strconv.Itoa(int(priorStateData.ID.ValueInt32()))),
					ProjectID:         types.StringValue(strconv.Itoa(int(priorStateData.ProjectID.ValueInt32()))),
					Name:              priorStateData.Name,
					Script:            priorStateData.Script,
					BaselineTestRunID: types.StringValue(strconv.Itoa(int(priorStateData.BaselineTestRunID.ValueInt32()))),
					Created:           priorStateData.Created,
					Updated:           priorStateData.Updated,
				}

				diags = resp.State.Set(ctx, upgradedStateData)
				resp.Diagnostics.Append(diags...)
			},
		},
	}
}

// Create creates the resource and sets the Terraform state on success.
func (r *loadTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan loadTestResourceModelV1
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := strconv.ParseInt(plan.ProjectID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project ID",
			"Could not parse project ID '"+plan.ProjectID.ValueString()+"': "+err.Error(),
		)
		return
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.LoadTestsAPI.ProjectsLoadTestsCreate(ctx, int32(projectID)).
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
	plan.ID = types.StringValue(strconv.Itoa(int(lt.GetId())))
	plan.Name = types.StringValue(lt.GetName())
	plan.ProjectID = types.StringValue(strconv.Itoa(int(lt.GetProjectId())))
	plan.BaselineTestRunID = handleBaselineTestRunID(lt.GetBaselineTestRunId())
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
	var state loadTestResourceModelV1
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the ID is empty, we cannot read the resource.
	// This is required for crossplane to work, but it never happens in Terraform in practice.
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	intID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing load test ID",
			"Could not parse load test ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	loadTestID := int32(intID)

	// Retrieve the load test attributes
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.LoadTestsAPI.LoadTestsRetrieve(ctx, loadTestID).
		XStackId(r.config.StackID)

	lt, httpResp, err := k6Req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 load test",
			"Could not read k6 load test with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Retrieve the load test script content
	scriptReq := r.client.LoadTestsAPI.LoadTestsScriptRetrieve(ctx, loadTestID).
		XStackId(r.config.StackID)

	script, httpResp, err := scriptReq.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		// 404 response from the script endpoint for an existing test means that the script is undefined
		script = ""
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 load test script",
			"Could not read k6 load test script with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.ID = types.StringValue(strconv.Itoa(int(lt.GetId())))
	state.Name = types.StringValue(lt.GetName())
	state.ProjectID = types.StringValue(strconv.Itoa(int(lt.GetProjectId())))
	state.BaselineTestRunID = handleBaselineTestRunID(lt.GetBaselineTestRunId())
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
	var plan loadTestResourceModelV1
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state to retrieve the ID
	var state loadTestResourceModelV1
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	intID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing load test ID",
			"Could not parse load test ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	loadTestID := int32(intID)

	// Generate API request body from plan
	toUpdate := k6.NewPatchLoadTestApiModel()
	toUpdate.SetName(plan.Name.ValueString())
	if plan.BaselineTestRunID.IsNull() {
		toUpdate.SetBaselineTestRunIdNil()
	} else {
		intID, err := strconv.ParseInt(plan.BaselineTestRunID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing baseline test run ID",
				"Could not parse baseline test run ID '"+state.BaselineTestRunID.ValueString()+"': "+err.Error(),
			)
			return
		}
		toUpdate.SetBaselineTestRunId(int32(intID))
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	updateReq := r.client.LoadTestsAPI.LoadTestsPartialUpdate(ctx, loadTestID).
		PatchLoadTestApiModel(toUpdate).
		XStackId(r.config.StackID)

	// Update the load test
	_, err = updateReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating k6 load test",
			"Could not update k6 load test with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update the script if it has changed
	if !plan.Script.Equal(state.Script) {
		scriptReq := r.client.LoadTestsAPI.LoadTestsScriptUpdate(ctx, loadTestID).
			Body(io.NopCloser(strings.NewReader(plan.Script.ValueString()))).
			XStackId(r.config.StackID)

		_, err = scriptReq.Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating k6 load test script",
				"Could not update k6 load test script with id "+state.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Update resource state with updated items and timestamp
	fetchReq := r.client.LoadTestsAPI.LoadTestsRetrieve(ctx, loadTestID).
		XStackId(r.config.StackID)

	lt, _, err := fetchReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 load test",
			"Could not read k6 load test with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Retrieve the updated script content
	scriptReq := r.client.LoadTestsAPI.LoadTestsScriptRetrieve(ctx, loadTestID).
		XStackId(r.config.StackID)

	script, _, err := scriptReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 load test script",
			"Could not read k6 load test script with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	plan.ID = types.StringValue(strconv.Itoa(int(lt.GetId())))
	plan.Name = types.StringValue(lt.GetName())
	plan.ProjectID = types.StringValue(strconv.Itoa(int(lt.GetProjectId())))
	plan.BaselineTestRunID = handleBaselineTestRunID(lt.GetBaselineTestRunId())
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
	var state loadTestResourceModelV1
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	intID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing load test ID",
			"Could not parse load test ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	loadTestID := int32(intID)

	// Delete existing load test
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	deleteReq := r.client.LoadTestsAPI.LoadTestsDestroy(ctx, loadTestID).
		XStackId(r.config.StackID)

	_, err = deleteReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting k6 load test",
			"Could not delete k6 load test with id "+strconv.Itoa(int(loadTestID))+": "+err.Error(),
		)
	}
}

func (r *loadTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
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
