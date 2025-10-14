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
	resourceFolderPermissionItemName = "grafana_folder_permission_item"
	resourceFolderPermissionItemID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("folderUID"), common.StringIDField("type (role, team, or user)"), common.StringIDField("identifier"))

	// Check interface
	_ resource.ResourceWithImportState = (*resourceFolderPermissionItem)(nil)
)

func makeResourceFolderPermissionItem() *common.Resource {
	resourceStruct := &resourceFolderPermissionItem{
		resourcePermissionBase: resourcePermissionBase{
			resourceType: foldersPermissionsType,
		},
	}
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceFolderPermissionItemName,
		resourceFolderPermissionItemID,
		resourceStruct,
	)
}

type resourceFolderPermissionItemModel struct {
	ID         types.String `tfsdk:"id"`
	OrgID      types.String `tfsdk:"org_id"`
	Role       types.String `tfsdk:"role"`
	Team       types.String `tfsdk:"team"`
	User       types.String `tfsdk:"user"`
	Permission types.String `tfsdk:"permission"`
	FolderUID  types.String `tfsdk:"folder_uid"`
}

// Framework doesn't support embedding a base struct: https://github.com/hashicorp/terraform-plugin-framework/issues/242
func (m *resourceFolderPermissionItemModel) ToBase() *resourcePermissionItemBaseModel {
	return &resourcePermissionItemBaseModel{
		ID:         m.ID,
		OrgID:      m.OrgID,
		Role:       m.Role,
		Team:       m.Team,
		User:       m.User,
		Permission: m.Permission,
	}
}

func (m *resourceFolderPermissionItemModel) SetFromBase(base *resourcePermissionItemBaseModel) {
	m.FolderUID = base.ResourceID
	m.ID = base.ID
	m.OrgID = base.OrgID
	m.Role = base.Role
	m.Team = base.Team
	m.User = base.User
	m.Permission = base.Permission
}

type resourceFolderPermissionItem struct{ resourcePermissionBase }

func (r *resourceFolderPermissionItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceFolderPermissionItemName
}

func (r *resourceFolderPermissionItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a single permission item for a folder. Conflicts with the "grafana_folder_permission" resource which manages the entire set of permissions for a folder.
		* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
		* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_permissions/)`,
		Attributes: r.addInSchemaAttributes(map[string]schema.Attribute{
			"folder_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the folder.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		}),
	}
}

func (r *resourceFolderPermissionItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.readItem(req.ID, r.folderQuery)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	var data resourceFolderPermissionItemModel
	data.SetFromBase(readData)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermissionItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.FolderUID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermissionItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	readData, diags := r.readItem(data.ID.ValueString(), r.folderQuery)
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

func (r *resourceFolderPermissionItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := data.ToBase()
	if diags := r.writeItem(data.FolderUID.ValueString(), base); diags != nil {
		resp.Diagnostics = diags
		return
	}
	data.SetFromBase(base)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermissionItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	data.Permission = types.StringValue("")

	if diags := r.writeItem(data.FolderUID.ValueString(), data.ToBase()); diags != nil {
		resp.Diagnostics = diags
	}
}

func (r *resourceFolderPermissionItem) folderQuery(client *client.GrafanaHTTPAPI, folderUID string) error {
	_, err := client.Folders.GetFolderByUID(folderUID)
	return err
}
