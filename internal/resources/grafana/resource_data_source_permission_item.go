package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
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
	return common.NewResource(resourceDatasourcePermissionItemName, resourceDatasourcePermissionItemID, resourceStruct)
}

type resourceDatasourcePermissionItemModel struct {
	resourceFolderPermissionItemModel
	DatasourceUID types.String `tfsdk:"datasource_uid"`
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
	data, diags := r.readItem(req.ID, r.datasourceQuery)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if data == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDatasourcePermissionItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if diags := r.writeItem(data.DatasourceUID.ValueString(), &data.resourcePermissionItemBaseModel); diags != nil {
		resp.Diagnostics = diags
		return
	}

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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *resourceDatasourcePermissionItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if diags := r.writeItem(data.DatasourceUID.ValueString(), &data.resourcePermissionItemBaseModel); diags != nil {
		resp.Diagnostics = diags
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceDatasourcePermissionItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	data.Permission = types.StringValue("")

	if diags := r.writeItem(data.DatasourceUID.ValueString(), &data.resourcePermissionItemBaseModel); diags != nil {
		resp.Diagnostics = diags
	}
}

func (r *resourceDatasourcePermissionItem) datasourceQuery(client *client.GrafanaHTTPAPI, datasourceUID string) error {
	_, err := client.Datasources.GetDataSourceByUID(datasourceUID)
	return err
}
