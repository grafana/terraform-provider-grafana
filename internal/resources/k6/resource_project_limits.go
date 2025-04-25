package k6

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithConfigure   = (*projectLimitsResource)(nil)
	_ resource.ResourceWithImportState = (*projectLimitsResource)(nil)
)

var (
	resourceProjectLimitsName = "grafana_k6_project_limits"
	resourceProjectLimitsID   = common.NewResourceID(common.IntIDField("project_id"))
)

func resourceProjectLimits() *common.Resource {
	return common.NewResource(
		common.CategoryK6,
		resourceProjectLimitsName,
		resourceProjectLimitsID,
		&projectLimitsResource{},
	)
}

// projectLimitsResourceModel maps the resource schema data.
type projectLimitsResourceModel struct {
	ID                  types.Int32 `tfsdk:"id"`
	ProjectID           types.Int32 `tfsdk:"project_id"`
	VuhMaxPerMonth      types.Int32 `tfsdk:"vuh_max_per_month"`
	VuMaxPerTest        types.Int32 `tfsdk:"vu_max_per_test"`
	VuBrowserMaxPerTest types.Int32 `tfsdk:"vu_browser_max_per_test"`
	DurationMaxPerTest  types.Int32 `tfsdk:"duration_max_per_test"`
}

// projectLimitsResource is the resource implementation.
type projectLimitsResource struct {
	basePluginFrameworkResource
}

// Metadata returns the resource type name.
func (r *projectLimitsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceProjectLimitsName
}

// Schema defines the schema for the resource.
func (r *projectLimitsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages limits for a k6 project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Description: "The identifier of the project limits. This is the same as the project_id.",
				Computed:    true,
			},
			"project_id": schema.Int32Attribute{
				Description: "The identifier of the project to manage limits for.",
				Required:    true,
			},
			"vuh_max_per_month": schema.Int32Attribute{
				Description: "Maximum amount of virtual user hours (VU/h) used per one calendar month.",
				Optional:    true,
			},
			"vu_max_per_test": schema.Int32Attribute{
				Description: "Maximum number of concurrent virtual users (VUs) used in one test.",
				Optional:    true,
			},
			"vu_browser_max_per_test": schema.Int32Attribute{
				Description: "Maximum number of concurrent browser virtual users (VUs) used in one test.",
				Optional:    true,
			},
			"duration_max_per_test": schema.Int32Attribute{
				Description: "Maximum duration of a test in seconds.",
				Optional:    true,
			},
		},
	}
}

// Create creates the resource and sets the Terraform state on success.
func (r *projectLimitsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectLimitsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the project_id
	plan.ID = plan.ProjectID

	// Generate API request body from plan
	toUpdate := k6.NewPatchProjectLimitsRequest()
	if !plan.VuhMaxPerMonth.IsNull() {
		toUpdate.SetVuhMaxPerMonth(plan.VuhMaxPerMonth.ValueInt32())
	}
	if !plan.VuMaxPerTest.IsNull() {
		toUpdate.SetVuMaxPerTest(plan.VuMaxPerTest.ValueInt32())
	}
	if !plan.VuBrowserMaxPerTest.IsNull() {
		toUpdate.SetVuBrowserMaxPerTest(plan.VuBrowserMaxPerTest.ValueInt32())
	}
	if !plan.DurationMaxPerTest.IsNull() {
		toUpdate.SetDurationMaxPerTest(plan.DurationMaxPerTest.ValueInt32())
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsLimitsPartialUpdate(ctx, plan.ProjectID.ValueInt32()).
		PatchProjectLimitsRequest(toUpdate).
		XStackId(r.config.StackID)

	// Update project limits
	_, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating k6 project limits",
			"Could not update k6 project limits, unexpected error: "+err.Error(),
		)
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read retrieves the resource information.
func (r *projectLimitsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state projectLimitsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the project_id
	state.ID = state.ProjectID

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsLimitsRetrieve(ctx, state.ProjectID.ValueInt32()).
		XStackId(r.config.StackID)

	limits, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 project limits",
			"Could not read k6 project limits for project with id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.VuhMaxPerMonth = types.Int32Value(limits.GetVuhMaxPerMonth())
	state.VuMaxPerTest = types.Int32Value(limits.GetVuMaxPerTest())
	state.VuBrowserMaxPerTest = types.Int32Value(limits.GetVuBrowserMaxPerTest())
	state.DurationMaxPerTest = types.Int32Value(limits.GetDurationMaxPerTest())

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectLimitsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan projectLimitsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the project_id
	plan.ID = plan.ProjectID

	// Generate API request body from plan
	toUpdate := k6.NewPatchProjectLimitsRequest()
	if !plan.VuhMaxPerMonth.IsNull() {
		toUpdate.SetVuhMaxPerMonth(plan.VuhMaxPerMonth.ValueInt32())
	}
	if !plan.VuMaxPerTest.IsNull() {
		toUpdate.SetVuMaxPerTest(plan.VuMaxPerTest.ValueInt32())
	}
	if !plan.VuBrowserMaxPerTest.IsNull() {
		toUpdate.SetVuBrowserMaxPerTest(plan.VuBrowserMaxPerTest.ValueInt32())
	}
	if !plan.DurationMaxPerTest.IsNull() {
		toUpdate.SetDurationMaxPerTest(plan.DurationMaxPerTest.ValueInt32())
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsLimitsPartialUpdate(ctx, plan.ProjectID.ValueInt32()).
		PatchProjectLimitsRequest(toUpdate).
		XStackId(r.config.StackID)

	// Update project limits
	_, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating k6 project limits",
			"Could not update k6 project limits for project with id "+strconv.Itoa(int(plan.ProjectID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectLimitsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state projectLimitsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset limits to default values
	toReset := k6.NewPatchProjectLimitsRequestWithDefaults()
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsLimitsPartialUpdate(ctx, state.ProjectID.ValueInt32()).
		PatchProjectLimitsRequest(toReset).
		XStackId(r.config.StackID)

	_, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error resetting k6 project limits",
			"Could not reset k6 project limits for project with id "+strconv.Itoa(int(state.ProjectID.ValueInt32()))+": "+err.Error(),
		)
	}
}

func (r *projectLimitsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing k6 project limits",
			"Could not parse k6 project id "+req.ID+": "+err.Error(),
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("project_id"), types.Int32Value(int32(id)))

	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}

	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
}
