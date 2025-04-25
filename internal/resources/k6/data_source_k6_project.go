package k6

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*projectDataSource)(nil)
)

var (
	dataSourceProjectName = "grafana_k6_project"
)

func dataSourceProject() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceProjectName,
		&projectDataSource{},
	)
}

// projectDataSourceModel maps the data source schema data.
type projectDataSourceModel struct {
	ID               types.Int32  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	IsDefault        types.Bool   `tfsdk:"is_default"`
	GrafanaFolderUid types.String `tfsdk:"grafana_folder_uid"`
	Created          types.String `tfsdk:"created"`
	Updated          types.String `tfsdk:"updated"`
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *projectDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceProjectName
}

// Schema defines the schema for the data source.
func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a k6 project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Description: "Numeric identifier of the project.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-friendly identifier of the project.",
				Computed:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Whether this project is the default for running tests when no explicit project identifier is provided.",
				Computed:    true,
			},
			"grafana_folder_uid": schema.StringAttribute{
				Description: "The Grafana folder uid.",
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: "The date when the project was created.",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "The date when the project was last updated.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.ProjectsAPI.ProjectsRetrieve(ctx, state.ID.ValueInt32()).
		XStackId(d.config.StackID)

	p, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 project",
			"Could not read k6 project with id "+strconv.Itoa(int(state.ID.ValueInt32()))+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(p.GetName())
	state.IsDefault = types.BoolValue(p.GetIsDefault())
	if p.GrafanaFolderUid.IsSet() {
		state.GrafanaFolderUid = types.StringValue(p.GetGrafanaFolderUid())
	} else {
		state.GrafanaFolderUid = types.StringNull()
	}
	state.Created = types.StringValue(p.GetCreated().Format(time.RFC3339Nano))
	state.Updated = types.StringValue(p.GetUpdated().Format(time.RFC3339Nano))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
