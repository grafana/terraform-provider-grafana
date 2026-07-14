package grafana

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana-openapi-client-go/client/library_elements"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var dataSourceLibraryPanelsName = "grafana_library_panels"

func datasourceLibraryPanels() *common.DataSource {
	return common.NewDataSource(
		common.CategoryGrafanaOSS,
		dataSourceLibraryPanelsName,
		&libraryPanelsDataSource{},
	)
}

type libraryPanelsDataSource struct {
	basePluginFrameworkDataSource
}

func (r *libraryPanelsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceLibraryPanelsName
}

func (r *libraryPanelsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"panels": schema.SetAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":        types.StringType,
						"uid":         types.StringType,
						"description": types.StringType,
						"folder_uid":  types.StringType,
						"model_json":  types.StringType,
					},
				},
			},
		},
	}
}

type libraryPanelsDataSourcePanelModel struct {
	Name        types.String `tfsdk:"name"`
	UID         types.String `tfsdk:"uid"`
	Description types.String `tfsdk:"description"`
	FolderUID   types.String `tfsdk:"folder_uid"`
	ModelJSON   types.String `tfsdk:"model_json"`
}

type libraryPanelsDataSourceModel struct {
	ID     types.String                        `tfsdk:"id"`
	OrgID  types.String                        `tfsdk:"org_id"`
	Panels []libraryPanelsDataSourcePanelModel `tfsdk:"panels"`
}

func (r *libraryPanelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data libraryPanelsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Read from API
	client, _, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to create client", err.Error())}
		return
	}
	params := library_elements.NewGetLibraryElementsParams().WithKind(common.Ref(libraryPanelKind))
	apiResp, err := client.LibraryElements.GetLibraryElements(params)
	if err != nil {
		resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get library panels", err.Error())}
		return
	}
	for _, panel := range apiResp.Payload.Result.Elements {
		modelJSONBytes, err := json.Marshal(panel.Model)
		if err != nil {
			resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get library panel JSON", err.Error())}
			return
		}
		data.Panels = append(data.Panels, libraryPanelsDataSourcePanelModel{
			Name:        types.StringValue(panel.Name),
			UID:         types.StringValue(panel.UID),
			Description: types.StringValue(panel.Description),
			FolderUID:   types.StringValue(panel.Meta.FolderUID),
			ModelJSON:   types.StringValue(string(modelJSONBytes)),
		})
	}
	data.ID = types.StringValue(data.OrgID.ValueString())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
