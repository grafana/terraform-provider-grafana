package grafana

import (
	"context"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

var (
	_ resource.ResourceWithImportState = (*resourceGroupAttributeMapping)(nil)

	resourceGroupAttributeMappingName = "grafana_group_attribute_mapping"
	resourceGroupAttributeMappingID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("group_id"))
)

func makeResourceGroupAttributeMapping() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceGroupAttributeMappingName,
		resourceGroupAttributeMappingID,
		&resourceGroupAttributeMapping{},
	).WithLister(listerFunctionOrgResource(listGroupAttributeMappings))
}

type resourceGroupAttributeMappingModel struct {
	ID       types.String   `tfsdk:"id"`
	OrgID    types.String   `tfsdk:"org_id"`
	GroupID  types.String   `tfsdk:"group_id"`
	RoleUIDs []types.String `tfsdk:"role_uids"`
}

type resourceGroupAttributeMapping struct {
	basePluginFrameworkResource
}

func (r *resourceGroupAttributeMapping) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceGroupAttributeMappingName
}

func (r *resourceGroupAttributeMapping) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Group attribute mapping is used to map groups from an external identity provider to Grafana attributes. This resource maps groups to fixed and custom role-based access control roles.

!> Warning: The resource is experimental and will be subject to change. To use the resource, you need to enable groupAttributeSync feature flag.

This resource requires Grafana Enterprise or Cloud >=11.4.0.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"group_id": schema.StringAttribute{
				Required:    true,
				Description: "Group ID from the identity provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_uids": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Fixed or custom Grafana role-based access control role UIDs.",
			},
		},
	}
}

func (r *resourceGroupAttributeMapping) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceGroupAttributeMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get client", err.Error())}
		return
	}

	roles := make([]string, 0, len(data.RoleUIDs))
	for _, roleUID := range data.RoleUIDs {
		roles = append(roles, roleUID.ValueString())
	}

	_, err = client.GroupAttributeSync.CreateGroupMappings(data.GroupID.ValueString(), &models.GroupAttributes{Roles: roles})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create group attribute mapping", err.Error())
		return
	}

	data.ID = types.StringValue(resourceGroupAttributeMappingID.Make(orgID, data.GroupID.ValueString()))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceGroupAttributeMapping) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceGroupAttributeMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(data.ID.ValueString())
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *resourceGroupAttributeMapping) read(id string) (*resourceGroupAttributeMappingModel, diag.Diagnostics) {
	client, orgID, idFields, err := r.clientFromExistingOrgResource(resourceGroupAttributeMappingID, id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get client", err.Error())}
	}
	groupID := idFields[0].(string)

	resp, err := client.GroupAttributeSync.GetGroupRoles(groupID)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get group mappings", err.Error())}
	}
	if resp.Payload == nil || len(resp.Payload) == 0 {
		return nil, nil
	}

	data := &resourceGroupAttributeMappingModel{
		ID:      types.StringValue(id),
		OrgID:   types.StringValue(strconv.FormatInt(orgID, 10)),
		GroupID: types.StringValue(groupID),
	}

	uids := make([]types.String, 0, len(resp.Payload))
	for _, role := range resp.Payload {
		uids = append(uids, types.StringValue(role.UID))
	}

	data.RoleUIDs = uids
	return data, nil
}

func (r *resourceGroupAttributeMapping) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceGroupAttributeMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, _, err := r.clientFromExistingOrgResource(resourceGroupAttributeMappingID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get client", err.Error())}
		return
	}

	roles := make([]string, 0, len(data.RoleUIDs))
	for _, roleUID := range data.RoleUIDs {
		roles = append(roles, roleUID.ValueString())
	}

	_, err = client.GroupAttributeSync.UpdateGroupMappings(data.GroupID.ValueString(), &models.GroupAttributes{Roles: roles})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create group attribute mapping", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceGroupAttributeMapping) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceGroupAttributeMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	client, _, idFields, err := r.clientFromExistingOrgResource(resourceGroupAttributeMappingID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	groupID := idFields[0].(string)

	if _, err := client.GroupAttributeSync.DeleteGroupMappings(groupID); err != nil {
		resp.Diagnostics.AddError("Unable to delete group mappings", err.Error())
	}
}

func (r *resourceGroupAttributeMapping) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data, diags := r.read(req.ID)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if data == nil {
		resp.Diagnostics.AddError("Group mapping not found", "Group mapping not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func listGroupAttributeMappings(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	var page int64 = 1
	for {
		resp, err := client.GroupAttributeSync.GetMappedGroups()
		if err != nil {
			return nil, err
		}
		for _, g := range resp.GetPayload().Groups {
			ids = append(ids, MakeOrgResourceID(orgID, g.GroupID))
		}

		if resp.Payload.Total <= int64(len(ids)) {
			break
		}

		page++
	}

	return ids, nil
}
