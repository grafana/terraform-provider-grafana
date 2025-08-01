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
	resourceDatasourcePermissionItemName = "grafana_data_source_permission_item"
	resourceDatasourcePermissionItemID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("datasourceUID"), common.StringIDField("type (role, team, or user)"), common.StringIDField("identifier"))

	// Check interface
	_ resource.ResourceWithImportState = (*resourceDatasourcePermissionItem)(nil)
)

func makeResourceDatasourcePermissionItem() *common.Resource {
	resourceStruct := &resourceDatasourcePermissionItem{
		resourcePermissionBase: resourcePermissionBase{
			resourceType: datasourcesPermissionsType,
		},
	}
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceDatasourcePermissionItemName,
		resourceDatasourcePermissionItemID,
		resourceStruct,
	)
}

type resourceDatasourcePermissionItemModel struct {
	ID            types.String `tfsdk:"id"`
	OrgID         types.String `tfsdk:"org_id"`
	Role          types.String `tfsdk:"role"`
	Team          types.String `tfsdk:"team"`
	User          types.String `tfsdk:"user"`
	Permission    types.String `tfsdk:"permission"`
	DatasourceUID types.String `tfsdk:"datasource_uid"`
}

// Framework doesn't support embedding a base struct: https://github.com/hashicorp/terraform-plugin-framework/issues/242
func (m *resourceDatasourcePermissionItemModel) ToBase() *resourcePermissionItemBaseModel {
	return &resourcePermissionItemBaseModel{
		ID:         m.ID,
		OrgID:      m.OrgID,
		Role:       m.Role,
		Team:       m.Team,
		User:       m.User,
		Permission: m.Permission,
	}
}

func (m *resourceDatasourcePermissionItemModel) SetFromBase(base *resourcePermissionItemBaseModel) {
	m.DatasourceUID = base.ResourceID
	m.ID = base.ID
	m.OrgID = base.OrgID
	m.Role = base.Role
	m.Team = base.Team
	m.User = base.User
	m.Permission = base.Permission
}

type resourceDatasourcePermissionItem struct{ resourcePermissionBase }

func (r *resourceDatasourcePermissionItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDatasourcePermissionItemName
}

func (r *resourceDatasourcePermissionItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a single permission item for a datasource. Conflicts with the "grafana_data_source_permission" resource which manages the entire set of permissions for a datasource.`,
		Attributes: r.addInSchemaAttributes(map[string]schema.Attribute{
			"datasource_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the datasource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		}),
	}
}

func (r *resourceDatasourcePermissionItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.readItem(req.ID, r.datasourceQuery)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	var data resourceDatasourcePermissionItemModel
	data.SetFromBase(readData)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDatasourcePermissionItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.DatasourceUID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDatasourcePermissionItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	readData, diags := r.readItem(data.ID.ValueString(), r.datasourceQuery)
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

func (r *resourceDatasourcePermissionItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.DatasourceUID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDatasourcePermissionItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	data.Permission = types.StringValue("")

	if diags := r.writeItem(data.DatasourceUID.ValueString(), data.ToBase()); diags != nil {
		resp.Diagnostics = diags
	}
}

func (r *resourceDatasourcePermissionItem) datasourceQuery(client *client.GrafanaHTTPAPI, datasourceUID string) error {
	_, err := client.Datasources.GetDataSourceByUID(datasourceUID)
	return err
}
