package k6

import (
	"context"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*projectsDataSource)(nil)
)

var (
	dataSourceProjectsName = "grafana_k6_projects"
)

func dataSourceProjects() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceProjectsName,
		&projectsDataSource{},
	)
}

// projectsDataSourceModel maps the data source schema data.
type projectsDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	Name     types.String             `tfsdk:"name"`
	Projects []projectDataSourceModel `tfsdk:"projects"`
}

// projectsDataSource is the data source implementation.
type projectsDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *projectsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceProjectsName
}

// Schema defines the schema for the data source.
func (d *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves all k6 projects with the given name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Human-friendly identifier of the project. This is the same as name.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the project.",
				Optional:    true,
			},
			"projects": schema.ListAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":                 types.Int32Type,
						"name":               types.StringType,
						"is_default":         types.BoolType,
						"grafana_folder_uid": types.StringType,
						"created":            types.StringType,
						"updated":            types.StringType,
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectsDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID to match the name
	state.ID = state.Name

	// Retrieve projects
	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.ProjectsAPI.ProjectsList(ctx).
		Name(state.Name.ValueString()).
		XStackId(d.config.StackID)

	pjs, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 projects",
			"Could not read k6 projects with name '"+state.Name.ValueString()+"': "+err.Error(),
		)
		return
	}

	// Process the results and populate the state with the retrieved projects
	var projectStates []projectDataSourceModel
	sort.Slice(pjs.Value, func(i, j int) bool {
		return pjs.Value[i].GetCreated().Before(pjs.Value[j].GetCreated())
	})
	for _, pj := range pjs.Value {
		// For each project, populate the state
		var grafanaFolderUid types.String
		if pj.GrafanaFolderUid.IsSet() {
			grafanaFolderUid = types.StringValue(pj.GetGrafanaFolderUid())
		} else {
			grafanaFolderUid = types.StringNull()
		}
		pjState := projectDataSourceModel{
			ID:               types.Int32Value(pj.GetId()),
			Name:             types.StringValue(pj.GetName()),
			IsDefault:        types.BoolValue(pj.GetIsDefault()),
			GrafanaFolderUid: grafanaFolderUid,
			Created:          types.StringValue(pj.GetCreated().Format(time.RFC3339Nano)),
			Updated:          types.StringValue(pj.GetUpdated().Format(time.RFC3339Nano)),
		}

		// Add the project state to the list
		projectStates = append(projectStates, pjState)
	}

	state.Projects = projectStates

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
