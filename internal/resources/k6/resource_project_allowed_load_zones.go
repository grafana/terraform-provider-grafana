package k6

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithConfigure   = (*projectAllowedLoadZonesResource)(nil)
	_ resource.ResourceWithImportState = (*projectAllowedLoadZonesResource)(nil)
)

var (
	resourceProjectAllowedLoadZonesName = "grafana_k6_project_allowed_load_zones"
	resourceProjectAllowedLoadZonesID   = common.NewResourceID(common.StringIDField("project_id"))
)

func resourceProjectAllowedLoadZones() *common.Resource {
	return common.NewResource(
		common.CategoryK6,
		resourceProjectAllowedLoadZonesName,
		resourceProjectAllowedLoadZonesID,
		&projectAllowedLoadZonesResource{},
	)
}

// projectAllowedLoadZonesResourceModel maps the resource schema data.
type projectAllowedLoadZonesResourceModel struct {
	ProjectID        types.Int32 `tfsdk:"project_id"`
	AllowedLoadZones types.List  `tfsdk:"allowed_load_zones"`
}

// projectAllowedLoadZonesResource is the resource implementation.
type projectAllowedLoadZonesResource struct {
	basePluginFrameworkResource
}

// Metadata returns the resource type name.
func (r *projectAllowedLoadZonesResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceProjectAllowedLoadZonesName
}

// Schema defines the schema for the resource.
func (r *projectAllowedLoadZonesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages allowed load zones for a k6 project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.Int32Attribute{
				Description: "The identifier of the project to manage allowed load zones for.",
				Required:    true,
			},
			"allowed_load_zones": schema.ListAttribute{
				Description: "List of allowed k6 load zone IDs for this project.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Create creates the resource and sets the Terraform state on success.
func (r *projectAllowedLoadZonesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectAllowedLoadZonesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueInt32()

	// Get load zones from plan
	var loadZones []string
	diags = plan.AllowedLoadZones.ElementsAs(ctx, &loadZones, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set allowed load zones
	err := setProjectAllowedLoadZones(ctx, r.client, r.config, projectID, loadZones)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting allowed load zones",
			"Could not set allowed load zones for k6 project: "+err.Error(),
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
func (r *projectAllowedLoadZonesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state projectAllowedLoadZonesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueInt32()

	// Get allowed load zones
	allowedZones, err := getProjectAllowedLoadZones(ctx, r.client, r.config, projectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading allowed load zones",
			"Could not read allowed load zones for k6 project: "+err.Error(),
		)
		return
	}

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

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectAllowedLoadZonesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan projectAllowedLoadZonesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueInt32()

	// Get load zones from plan
	var loadZones []string
	diags = plan.AllowedLoadZones.ElementsAs(ctx, &loadZones, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update allowed load zones
	err := setProjectAllowedLoadZones(ctx, r.client, r.config, projectID, loadZones)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating allowed load zones",
			"Could not update allowed load zones for k6 project: "+err.Error(),
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
func (r *projectAllowedLoadZonesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state projectAllowedLoadZonesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueInt32()

	// Clear allowed load zones (set to empty list)
	err := setProjectAllowedLoadZones(ctx, r.client, r.config, projectID, []string{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error clearing allowed load zones",
			"Could not clear allowed load zones for k6 project: "+err.Error(),
		)
	}
}

func (r *projectAllowedLoadZonesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
