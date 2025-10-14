package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceServiceAccountPermissionItemName = "grafana_service_account_permission_item"
	resourceServiceAccountPermissionItemID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("serviceAccountID"), common.StringIDField("type (role, team, or user)"), common.StringIDField("identifier"))

	// Check interface
	_ resource.ResourceWithImportState = (*resourceServiceAccountPermissionItem)(nil)
)

func makeResourceServiceAccountPermissionItem() *common.Resource {
	resourceStruct := &resourceServiceAccountPermissionItem{
		resourcePermissionBase: resourcePermissionBase{
			resourceType: serviceAccountsPermissionsType,
		},
	}
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceServiceAccountPermissionItemName,
		resourceServiceAccountPermissionItemID,
		resourceStruct,
	)
}

type resourceServiceAccountPermissionItemModel struct {
	ID               types.String `tfsdk:"id"`
	OrgID            types.String `tfsdk:"org_id"`
	Team             types.String `tfsdk:"team"`
	User             types.String `tfsdk:"user"`
	Permission       types.String `tfsdk:"permission"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
}

// Framework doesn't support embedding a base struct: https://github.com/hashicorp/terraform-plugin-framework/issues/242
func (m *resourceServiceAccountPermissionItemModel) ToBase() *resourcePermissionItemBaseModel {
	return &resourcePermissionItemBaseModel{
		ID:         m.ID,
		OrgID:      m.OrgID,
		Team:       m.Team,
		User:       m.User,
		Permission: m.Permission,
	}
}

func (m *resourceServiceAccountPermissionItemModel) SetFromBase(base *resourcePermissionItemBaseModel) {
	m.ServiceAccountID = base.ResourceID
	m.ID = base.ID
	m.OrgID = base.OrgID
	m.Team = base.Team
	m.User = base.User
	m.Permission = base.Permission
}

type resourceServiceAccountPermissionItem struct{ resourcePermissionBase }

func (r *resourceServiceAccountPermissionItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceServiceAccountPermissionItemName
}

func (r *resourceServiceAccountPermissionItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a single permission item for a service account. Conflicts with the "grafana_service_account_permission" resource which manages the entire set of permissions for a service account.
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/#manage-users-and-teams-permissions-for-a-service-account-in-grafana)`,
		Attributes: r.addInSchemaAttributes(map[string]schema.Attribute{
			"service_account_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the service account.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					&orgScopedAttributePlanModifier{},
				},
			},
		}),
	}

	// Role is not supported for service account permissions
	delete(resp.Schema.Attributes, permissionTargetRole)
	for key, attr := range resp.Schema.Attributes {
		if key != permissionTargetTeam && key != permissionTargetUser {
			continue
		}
		attrCast := attr.(schema.StringAttribute)
		attrCast.Validators = []validator.String{stringvalidator.ExactlyOneOf(
			path.MatchRoot(permissionTargetTeam),
			path.MatchRoot(permissionTargetUser),
		)}
		resp.Schema.Attributes[key] = attrCast
	}
}

func (r *resourceServiceAccountPermissionItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.readItem(req.ID, r.serviceAccountQuery)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	var data resourceServiceAccountPermissionItemModel
	data.SetFromBase(readData)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceServiceAccountPermissionItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceServiceAccountPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.ServiceAccountID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	base.ResourceID = data.ServiceAccountID
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceServiceAccountPermissionItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceServiceAccountPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	readData, diags := r.readItem(data.ID.ValueString(), r.serviceAccountQuery)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	readData.ResourceID = data.ServiceAccountID
	data.SetFromBase(readData)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceServiceAccountPermissionItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceServiceAccountPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.ServiceAccountID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	base.ResourceID = data.ServiceAccountID
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceServiceAccountPermissionItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceServiceAccountPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	data.Permission = types.StringValue("")

	if diags := r.writeItem(data.ServiceAccountID.ValueString(), data.ToBase()); diags != nil {
		resp.Diagnostics = diags
	}
}

func (r *resourceServiceAccountPermissionItem) serviceAccountQuery(client *client.GrafanaHTTPAPI, serviceAccountID string) error {
	idNumerical, err := strconv.ParseInt(serviceAccountID, 10, 64)
	if err != nil {
		return err
	}
	_, err = client.ServiceAccounts.RetrieveServiceAccount(idNumerical)
	return err
}
