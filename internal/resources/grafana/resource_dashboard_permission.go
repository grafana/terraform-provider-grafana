package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceDashboardPermissionName = "grafana_dashboard_permission"
	resourceDashboardPermissionID   = orgResourceIDString("dashboardUID")

	// Check interface
	_ resource.ResourceWithImportState = (*resourceDashboardPermission)(nil)
)

func makeResourceDashboardPermission() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceDashboardPermissionName,
		resourceDashboardPermissionID,
		&resourceDashboardPermission{
			resourcePermissionBulkBase: resourcePermissionBulkBase{
				resourceType: dashboardsPermissionsType,
			},
		},
	)
}

type resourceDashboardPermissionModel struct {
	ID           types.String              `tfsdk:"id"`
	OrgID        types.String              `tfsdk:"org_id"`
	DashboardUID types.String              `tfsdk:"dashboard_uid"`
	Permissions  []bulkPermissionItemModel `tfsdk:"permissions"`
}

type resourceDashboardPermission struct{ resourcePermissionBulkBase }

func (r *resourceDashboardPermission) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDashboardPermissionName
}

func (r *resourceDashboardPermission) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages the entire set of permissions for a dashboard. Permissions that aren't specified when applying this resource will be removed.
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard_permissions/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"dashboard_uid": schema.StringAttribute{
				Required:    true,
				Description: "UID of the dashboard to apply permissions to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"permissions": bulkPermissionsSchemaAttribute(
				"The permission items to add/update. Items that are omitted from the list will be removed.",
				[]string{"View", "Edit", "Admin"},
			),
		},
	}
}

func (r *resourceDashboardPermission) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, orgID, split, err := r.clientFromExistingOrgResource(resourceDashboardPermissionID, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse import ID", err.Error())
		return
	}
	dashboardUID := split[0].(string)

	_, err = client.Dashboards.GetDashboardByUID(dashboardUID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get dashboard", err.Error())
		return
	}

	permissions, diags := r.readBulkPermissions(client, dashboardUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := resourceDashboardPermissionModel{
		ID:           types.StringValue(MakeOrgResourceID(orgID, dashboardUID)),
		OrgID:        types.StringValue(strconv.FormatInt(orgID, 10)),
		DashboardUID: types.StringValue(dashboardUID),
		Permissions:  permissions,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermission) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceDashboardPermissionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	dashboardUID := data.DashboardUID.ValueString()
	_, err = client.Dashboards.GetDashboardByUID(dashboardUID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get dashboard", err.Error())
		return
	}

	resp.Diagnostics.Append(r.applyBulkPermissions(client, dashboardUID, data.Permissions)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions, diags := r.readBulkPermissions(client, dashboardUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, dashboardUID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.DashboardUID = types.StringValue(dashboardUID)
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermission) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceDashboardPermissionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, split, err := r.clientFromExistingOrgResource(resourceDashboardPermissionID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	dashboardUID := split[0].(string)

	_, err = client.Dashboards.GetDashboardByUID(dashboardUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get dashboard", err.Error())
		return
	}

	permissions, diags := r.readBulkPermissions(client, dashboardUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, dashboardUID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.DashboardUID = types.StringValue(dashboardUID)
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermission) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceDashboardPermissionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	dashboardUID := data.DashboardUID.ValueString()

	resp.Diagnostics.Append(r.applyBulkPermissions(client, dashboardUID, data.Permissions)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions, diags := r.readBulkPermissions(client, dashboardUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, dashboardUID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDashboardPermission) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceDashboardPermissionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceDashboardPermissionID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	dashboardUID := split[0].(string)

	resp.Diagnostics.Append(r.applyBulkPermissions(client, dashboardUID, nil)...)
}
