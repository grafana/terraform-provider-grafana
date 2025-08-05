package k6

import (
	"context"
	"net/http"
	"strconv"
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
	_ resource.ResourceWithConfigure    = (*projectResource)(nil)
	_ resource.ResourceWithImportState  = (*projectResource)(nil)
	_ resource.ResourceWithUpgradeState = (*projectResource)(nil)
)

var (
	resourceProjectName = "grafana_k6_project"
	resourceProjectID   = common.NewResourceID(common.StringIDField("id"))
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
type projectResourceModelV0 struct {
	ID               types.Int32  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	IsDefault        types.Bool   `tfsdk:"is_default"`
	GrafanaFolderUID types.String `tfsdk:"grafana_folder_uid"`
	Created          types.String `tfsdk:"created"`
	Updated          types.String `tfsdk:"updated"`
}

type projectResourceModelV1 struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	IsDefault        types.Bool   `tfsdk:"is_default"`
	GrafanaFolderUID types.String `tfsdk:"grafana_folder_uid"`
	Created          types.String `tfsdk:"created"`
	Updated          types.String `tfsdk:"updated"`
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
		Description: "Manages a k6 project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the project.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the project.",
				Required:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Use this project as default for running tests when no explicit project identifier is provided.",
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
		},
		Version: 1,
	}
}

func (r *projectResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.Int32Attribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Required: true,
					},
					"is_default": schema.BoolAttribute{
						Computed: true,
					},
					"grafana_folder_uid": schema.StringAttribute{
						Computed: true,
					},
					"created": schema.StringAttribute{
						Computed: true,
					},
					"updated": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// Convert int32 ID to string ID
				var priorStateData projectResourceModelV0
				diags := req.State.Get(ctx, &priorStateData)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := projectResourceModelV1{
					ID:               types.StringValue(strconv.Itoa(int(priorStateData.ID.ValueInt32()))),
					Name:             priorStateData.Name,
					IsDefault:        priorStateData.IsDefault,
					GrafanaFolderUID: priorStateData.GrafanaFolderUID,
					Created:          priorStateData.Created,
					Updated:          priorStateData.Updated,
				}

				diags = resp.State.Set(ctx, upgradedStateData)
				resp.Diagnostics.Append(diags...)
			},
		},
	}
}

// Create creates the resource and sets the Terraform state on success.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectResourceModelV1
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
			"Error creating k6 project",
			"Could not create k6 project, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.Itoa(int(p.GetId())))
	plan.Name = types.StringValue(p.GetName())
	plan.IsDefault = types.BoolValue(p.GetIsDefault())
	plan.GrafanaFolderUID = handleGrafanaFolderUID(p.GrafanaFolderUid)
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
	var state projectResourceModelV1
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the ID is empty, than it must be a call from crossplane during reconciliation of a new resource.
	// This is required for crossplane to work when Read is called before Create, but it never happens in Terraform in practice.
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	projectID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project ID",
			"Could not parse project ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}

	k6Req := r.client.ProjectsAPI.ProjectsRetrieve(ctx, int32(projectID)).
		XStackId(r.config.StackID)

	p, httpResp, err := k6Req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 project",
			"Could not read k6 project with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.ID = types.StringValue(strconv.Itoa(int(p.GetId())))
	state.Name = types.StringValue(p.GetName())
	state.IsDefault = types.BoolValue(p.GetIsDefault())
	state.GrafanaFolderUID = handleGrafanaFolderUID(p.GrafanaFolderUid)
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
	var plan projectResourceModelV1
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state to retrieve the ID
	var state projectResourceModelV1
	diags = req.State.Get(ctx, &state)
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

	// Generate API request body from plan
	toUpdate := k6.NewPatchProjectApiModel(plan.Name.ValueString())

	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	updateReq := r.client.ProjectsAPI.ProjectsPartialUpdate(ctx, projectID).
		PatchProjectApiModel(toUpdate).
		XStackId(r.config.StackID)

	// Update the project
	_, err = updateReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating k6 project",
			"Could not update k6 project with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update resource state with updated items and timestamp
	fetchReq := r.client.ProjectsAPI.ProjectsRetrieve(ctx, projectID).
		XStackId(r.config.StackID)

	p, _, err := fetchReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 project",
			"Could not read k6 project with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	plan.ID = types.StringValue(strconv.Itoa(int(p.GetId())))
	plan.Name = types.StringValue(p.GetName())
	plan.IsDefault = types.BoolValue(p.GetIsDefault())
	plan.GrafanaFolderUID = handleGrafanaFolderUID(p.GrafanaFolderUid)
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
	var state projectResourceModelV1
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

	// Delete existing project
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	deleteReq := r.client.ProjectsAPI.ProjectsDestroy(ctx, projectID).
		XStackId(r.config.StackID)

	_, err = deleteReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting k6 project",
			"Could not delete k6 project with id "+state.ID.ValueString()+": "+err.Error(),
		)
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
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
