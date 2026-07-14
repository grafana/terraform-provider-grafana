package k6

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithConfigure    = (*projectLimitsResource)(nil)
	_ resource.ResourceWithImportState  = (*projectLimitsResource)(nil)
	_ resource.ResourceWithUpgradeState = (*projectLimitsResource)(nil)
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
type projectLimitsResourceModelV0 struct {
	ID                  types.Int32 `tfsdk:"id"`
	ProjectID           types.Int32 `tfsdk:"project_id"`
	VuhMaxPerMonth      types.Int32 `tfsdk:"vuh_max_per_month"`
	VuMaxPerTest        types.Int32 `tfsdk:"vu_max_per_test"`
	VuBrowserMaxPerTest types.Int32 `tfsdk:"vu_browser_max_per_test"`
	DurationMaxPerTest  types.Int32 `tfsdk:"duration_max_per_test"`
}

type projectLimitsResourceModelV1 struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	VuhMaxPerMonth      types.Int32  `tfsdk:"vuh_max_per_month"`
	VuMaxPerTest        types.Int32  `tfsdk:"vu_max_per_test"`
	VuBrowserMaxPerTest types.Int32  `tfsdk:"vu_browser_max_per_test"`
	DurationMaxPerTest  types.Int32  `tfsdk:"duration_max_per_test"`
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
			"id": schema.StringAttribute{
				Description: "The identifier of the project limits. This is the same as the project_id.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The identifier of the project to manage limits for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
		Version: 1,
	}
}

func (r *projectLimitsResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
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
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// Convert int32 ID to string ID
				var priorStateData projectLimitsResourceModelV0
				diags := req.State.Get(ctx, &priorStateData)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := projectLimitsResourceModelV1{
					ID:                  types.StringValue(strconv.Itoa(int(priorStateData.ID.ValueInt32()))),
					ProjectID:           types.StringValue(strconv.Itoa(int(priorStateData.ProjectID.ValueInt32()))),
					VuhMaxPerMonth:      priorStateData.VuhMaxPerMonth,
					VuMaxPerTest:        priorStateData.VuMaxPerTest,
					VuBrowserMaxPerTest: priorStateData.VuBrowserMaxPerTest,
					DurationMaxPerTest:  priorStateData.DurationMaxPerTest,
				}

				diags = resp.State.Set(ctx, upgradedStateData)
				resp.Diagnostics.Append(diags...)
			},
		},
	}
}

// Create creates the resource and sets the Terraform state on success.
func (r *projectLimitsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectLimitsResourceModelV1
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the project_id
	plan.ID = plan.ProjectID

	intID, err := strconv.ParseInt(plan.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project ID",
			"Could not parse project ID '"+plan.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	projectID := int32(intID)

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
	k6Req := r.client.ProjectsAPI.ProjectsLimitsPartialUpdate(ctx, projectID).
		PatchProjectLimitsRequest(toUpdate).
		XStackId(r.config.StackID)

	// Update project limits
	_, err = k6Req.Execute()
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
	var state projectLimitsResourceModelV1
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the project_id
	state.ID = state.ProjectID

	intID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project ID",
			"Could not parse project ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	projectID := int32(intID)

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsLimitsRetrieve(ctx, projectID).
		XStackId(r.config.StackID)

	limits, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 project limits",
			"Could not read k6 project limits for project with id "+state.ID.ValueString()+": "+err.Error(),
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
	var plan projectLimitsResourceModelV1
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the project_id
	plan.ID = plan.ProjectID

	intID, err := strconv.ParseInt(plan.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project ID",
			"Could not parse project ID '"+plan.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	projectID := int32(intID)

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
	k6Req := r.client.ProjectsAPI.ProjectsLimitsPartialUpdate(ctx, projectID).
		PatchProjectLimitsRequest(toUpdate).
		XStackId(r.config.StackID)

	// Update project limits
	_, err = k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating k6 project limits",
			"Could not update k6 project limits for project with id "+plan.ID.ValueString()+": "+err.Error(),
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
	var state projectLimitsResourceModelV1
	diags := req.State.Get(ctx, &state)
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

	// Reset limits to default values
	toReset := k6.NewPatchProjectLimitsRequestWithDefaults()
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsLimitsPartialUpdate(ctx, projectID).
		PatchProjectLimitsRequest(toReset).
		XStackId(r.config.StackID)

	_, err = k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error resetting k6 project limits",
			"Could not reset k6 project limits for project with id "+state.ID.ValueString()+": "+err.Error(),
		)
	}
}

func (r *projectLimitsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
