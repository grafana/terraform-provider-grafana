package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceDashboardPermissionItemName = "grafana_dashboard_permission_item"
	resourceDashboardPermissionItemID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("dashboardUID"), common.StringIDField("type (role, team, or user)"), common.StringIDField("identifier"))

	// Check interface
	_ resource.ResourceWithImportState = (*resourceDashboardPermissionItem)(nil)
)

func makeResourceDashboardPermissionItem() *common.Resource {
	resourceStruct := &resourceDashboardPermissionItem{
		resourcePermissionBase: resourcePermissionBase{
			resourceType: dashboardsPermissionsType,
		},
	}
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceDashboardPermissionItemName,
		resourceDashboardPermissionItemID,
		resourceStruct,
	)
}

type resourceDashboardPermissionItemModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Role         types.String `tfsdk:"role"`
	Team         types.String `tfsdk:"team"`
	User         types.String `tfsdk:"user"`
	Permission   types.String `tfsdk:"permission"`
	DashboardUID types.String `tfsdk:"dashboard_uid"`
}

// Framework doesn't support embedding a base struct: https://github.com/hashicorp/terraform-plugin-framework/issues/242
func (m *resourceDashboardPermissionItemModel) ToBase() *resourcePermissionItemBaseModel {
	return &resourcePermissionItemBaseModel{
		ID:         m.ID,
		OrgID:      m.OrgID,
		Role:       m.Role,
		Team:       m.Team,
		User:       m.User,
		Permission: m.Permission,
	}
}

func (m *resourceDashboardPermissionItemModel) SetFromBase(base *resourcePermissionItemBaseModel) {
	m.DashboardUID = base.ResourceID
	m.ID = base.ID
	m.OrgID = base.OrgID
	m.Role = base.Role
	m.Team = base.Team
	m.User = base.User
	m.Permission = base.Permission
}

type resourceDashboardPermissionItem struct{ resourcePermissionBase }

func (r *resourceDashboardPermissionItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDashboardPermissionItemName
}

func (r *resourceDashboardPermissionItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a single permission item for a dashboard. Conflicts with the "grafana_dashboard_permission" resource which manages the entire set of permissions for a dashboard.`,
		Attributes: r.addInSchemaAttributes(map[string]schema.Attribute{
			"dashboard_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		}),
	}
}

func (r *resourceDashboardPermissionItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.readItem(req.ID, r.dashboardQuery)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	var data resourceDashboardPermissionItemModel
	data.SetFromBase(readData)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermissionItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceDashboardPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.DashboardUID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermissionItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceDashboardPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	readData, diags := r.readItem(data.ID.ValueString(), r.dashboardQuery)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.SetFromBase(readData)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermissionItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceDashboardPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.DashboardUID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermissionItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceDashboardPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	data.Permission = types.StringValue("")

	if diags := r.writeItem(data.DashboardUID.ValueString(), data.ToBase()); diags != nil {
		resp.Diagnostics = diags
	}
}

func (r *resourceDashboardPermissionItem) dashboardQuery(client *client.GrafanaHTTPAPI, dashboardUID string) error {
	_, err := client.Dashboards.GetDashboardByUID(dashboardUID)
	return err
}
