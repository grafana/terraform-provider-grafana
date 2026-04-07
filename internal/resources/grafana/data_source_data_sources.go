package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*dataSourcesDataSource)(nil)

func datasourceDataSources() *common.DataSource {
	return common.NewDataSource(
		common.CategoryGrafanaOSS,
		"grafana_data_sources",
		&dataSourcesDataSource{},
	)
}

type dataSourcesDataSourceModel struct {
	ID          types.String                     `tfsdk:"id"`
	OrgID       types.String                     `tfsdk:"org_id"`
	DataSources []dataSourcesDataSourceItemModel `tfsdk:"data_sources"`
}

type dataSourcesDataSourceItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	UID       types.String `tfsdk:"uid"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	URL       types.String `tfsdk:"url"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	Access    types.String `tfsdk:"access"`
	Database  types.String `tfsdk:"database"`
	BasicAuth types.Bool   `tfsdk:"basic_auth"`
	User      types.String `tfsdk:"user"`
}

type dataSourcesDataSource struct {
	basePluginFrameworkDataSource
}

func (d *dataSourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "grafana_data_sources"
}

func (d *dataSourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/data_source/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"data_sources": schema.ListAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":         types.Int64Type,
						"uid":        types.StringType,
						"name":       types.StringType,
						"type":       types.StringType,
						"url":        types.StringType,
						"is_default": types.BoolType,
						"access":     types.StringType,
						"database":   types.StringType,
						"basic_auth": types.BoolType,
						"user":       types.StringType,
					},
				},
				Description: "A list of Grafana data sources.",
			},
		},
	}
}

func (d *dataSourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dataSourcesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := d.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	getResp, err := client.Datasources.GetDataSources()
	if err != nil {
		resp.Diagnostics.AddError("Failed to get data sources", err.Error())
		return
	}

	var allDataSources []dataSourcesDataSourceItemModel

	for _, ds := range getResp.GetPayload() {
		allDataSources = append(allDataSources, dataSourcesDataSourceItemModel{
			ID:        types.Int64Value(ds.ID),
			UID:       types.StringValue(ds.UID),
			Name:      types.StringValue(ds.Name),
			Type:      types.StringValue(ds.Type),
			URL:       types.StringValue(ds.URL),
			IsDefault: types.BoolValue(ds.IsDefault),
			Access:    types.StringValue(string(ds.Access)),
			Database:  types.StringValue(ds.Database),
			BasicAuth: types.BoolValue(ds.BasicAuth),
			User:      types.StringValue(ds.User),
		})
	}

	data.ID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.DataSources = allDataSources

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
