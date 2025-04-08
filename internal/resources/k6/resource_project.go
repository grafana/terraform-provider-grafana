package k6

import (
	"context"
	"strconv"
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
	_ resource.ResourceWithConfigure   = (*projectResource)(nil)
	_ resource.ResourceWithImportState = (*projectResource)(nil)
)

var (
	resourceProjectName = "grafana_k6_project"
	resourceProjectID   = common.NewResourceID(common.IntIDField("id"))
)

func resourceProject() *common.Resource {
	return common.NewResource(
		common.CategoryK6,
		resourceProjectName,
		resourceProjectID,
		&projectResource{},
	).WithLister(k6ListerFunction(listProjects))
}

// projectResourceModel maps the resource schema data.
type projectResourceModel struct {
	ID        types.Int32  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	Created   types.String `tfsdk:"created"`
	Updated   types.String `tfsdk:"updated"`
}

// projectResource is the resource implementation.
type projectResource struct {
	basePluginFrameworkResource
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceProjectName
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Description: "Numeric identifier of the project.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the project.",
				Required:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Use this project as default for running tests when no explicit project ID is provided.",
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
		},
	}
}

// Create creates the resource and sets the Terraform state on success.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	toCreate := k6.NewCreateProjectApiModel(plan.Name.ValueString())

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsCreate(ctx).
		CreateProjectApiModel(toCreate).
		XStackId(r.config.StackID)

	// Create new project
	p, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GCk6 project",
			"Could not create GCk6 project, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.Int32Value(p.GetId())
	plan.Name = types.StringValue(p.GetName())
	plan.IsDefault = types.BoolValue(p.GetIsDefault())
	plan.Created = types.StringValue(p.GetCreated().Format(time.RFC3339Nano))
	plan.Updated = types.StringValue(p.GetUpdated().Format(time.RFC3339Nano))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read retrieves the resource information.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	k6Req := r.client.ProjectsAPI.ProjectsRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	p, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading GCk6 project",
			"Could not read GCk6 project id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.ID = types.Int32Value(p.GetId())
	state.Name = types.StringValue(p.GetName())
	state.IsDefault = types.BoolValue(p.GetIsDefault())
	state.Created = types.StringValue(p.GetCreated().Format(time.RFC3339Nano))
	state.Updated = types.StringValue(p.GetUpdated().Format(time.RFC3339Nano))

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state to retrieve the ID
	var state projectResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	toUpdate := k6.NewPatchProjectApiModel(plan.Name.ValueString())

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	updateReq := r.client.ProjectsAPI.ProjectsPartialUpdate(ctx, state.ID.ValueInt32()).
		PatchProjectApiModel(toUpdate).
		XStackId(r.config.StackID)

	// Update the project
	_, err := updateReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating GCk6 project",
			"Could not update GCk6 project, unexpected error: "+err.Error(),
		)
		return
	}

	// Update resource state with updated items and timestamp
	fetchReq := r.client.ProjectsAPI.ProjectsRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	p, _, err := fetchReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading GCk6 project",
			"Could not read GCk6 project id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	plan.ID = types.Int32Value(p.GetId())
	plan.Name = types.StringValue(p.GetName())
	plan.IsDefault = types.BoolValue(p.GetIsDefault())
	plan.Created = types.StringValue(p.GetCreated().Format(time.RFC3339Nano))
	plan.Updated = types.StringValue(p.GetUpdated().Format(time.RFC3339Nano))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing project
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	deleteReq := r.client.ProjectsAPI.ProjectsDestroy(ctx, state.ID.ValueInt32()).
		XStackId(r.config.StackID)

	_, err := deleteReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting GCk6 project",
			"Could not delete GCk6 project id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing GCk6 project",
			"Could not parse GCk6 project id "+req.ID+": "+err.Error(),
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("id"), types.Int32Value(int32(id)))

	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}

	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
}

// listProjects retrieves the list ids of all the existing projects.
func listProjects(ctx context.Context, client *k6.APIClient, config *k6providerapi.K6APIConfig) ([]string, error) {
	ctx = context.WithValue(ctx, k6.ContextAccessToken, config.Token)
	resp, _, err := client.ProjectsAPI.ProjectsList(ctx).
		XStackId(config.StackID).
		Execute()
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, p := range resp.Value {
		ids = append(ids, strconv.Itoa(int(p.GetId())))
	}
	return ids, nil
}
