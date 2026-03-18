package grafana

import (
	"context"
	"regexp"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceFolderPermissionName = "grafana_folder_permission"
	resourceFolderPermissionID   = orgResourceIDString("folderUID")

	// Check interface
	_ resource.ResourceWithImportState = (*resourceFolderPermission)(nil)
)

func makeResourceFolderPermission() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceFolderPermissionName,
		resourceFolderPermissionID,
		&resourceFolderPermission{
			resourcePermissionBulkBase: resourcePermissionBulkBase{
				resourceType: foldersPermissionsType,
			},
		},
	)
}

type resourceFolderPermissionModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	FolderUID   types.String `tfsdk:"folder_uid"`
	Permissions types.Set    `tfsdk:"permissions"`
}

type resourceFolderPermission struct{ resourcePermissionBulkBase }

func (r *resourceFolderPermission) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceFolderPermissionName
}

func (r *resourceFolderPermission) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages the entire set of permissions for a folder. Permissions that aren't specified when applying this resource will be removed.
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_permissions/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"folder_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the folder.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9\-\_]+$`), "folder UIDs can only be alphanumeric, dashes, or underscores"),
				},
			},
			"permissions": bulkPermissionsSchemaAttribute(
				"The permission items to add/update. Items that are omitted from the list will be removed.",
				[]string{"View", "Edit", "Admin"},
			),
		},
	}
}

func (r *resourceFolderPermission) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, orgID, split, err := r.clientFromExistingOrgResource(resourceFolderPermissionID, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse import ID", err.Error())
		return
	}
	folderUID := split[0].(string)

	folderResp, err := client.Folders.GetFolderByUID(folderUID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get folder", err.Error())
		return
	}
	folderUID = folderResp.Payload.UID

	permissions, diags := r.readBulkPermissions(ctx, client, folderUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := resourceFolderPermissionModel{
		ID:          types.StringValue(MakeOrgResourceID(orgID, folderUID)),
		OrgID:       types.StringValue(strconv.FormatInt(orgID, 10)),
		FolderUID:   types.StringValue(folderUID),
		Permissions: permissions,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermission) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceFolderPermissionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	folderResp, err := client.Folders.GetFolderByUID(data.FolderUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get folder", err.Error())
		return
	}
	folderUID := folderResp.Payload.UID

	resp.Diagnostics.Append(r.applyBulkPermissions(ctx, client, folderUID, data.Permissions)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions, diags := r.readBulkPermissions(ctx, client, folderUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, folderUID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.FolderUID = types.StringValue(folderUID)
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermission) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceFolderPermissionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, split, err := r.clientFromExistingOrgResource(resourceFolderPermissionID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	folderUID := split[0].(string)

	folderResp, err := client.Folders.GetFolderByUID(folderUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get folder", err.Error())
		return
	}
	folderUID = folderResp.Payload.UID

	permissions, diags := r.readBulkPermissions(ctx, client, folderUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, folderUID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.FolderUID = types.StringValue(folderUID)
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermission) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceFolderPermissionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	folderUID := data.FolderUID.ValueString()

	resp.Diagnostics.Append(r.applyBulkPermissions(ctx, client, folderUID, data.Permissions)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions, diags := r.readBulkPermissions(ctx, client, folderUID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, folderUID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermission) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceFolderPermissionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceFolderPermissionID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	folderUID := split[0].(string)

	emptyPerms := types.SetValueMust(types.ObjectType{AttrTypes: bulkPermissionItemAttrTypes}, nil)
	resp.Diagnostics.Append(r.applyBulkPermissions(ctx, client, folderUID, emptyPerms)...)
}
