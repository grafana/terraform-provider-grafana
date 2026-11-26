package grafana

import (
	"context"
	"sort"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &dataSourcesDataSource{}

type dataSourcesDataSource struct {
	basePluginFrameworkDataSource
}

type dataSourcesDataSourceModel struct {
	ID          types.String                 `tfsdk:"id"`
	OrgID       types.String                 `tfsdk:"org_id"`
	DataSources []dataSourceListElementModel `tfsdk:"data_sources"`
}

type dataSourceListElementModel struct {
	ID        types.Int64  `tfsdk:"id"`
	UID       types.String `tfsdk:"uid"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	URL       types.String `tfsdk:"url"`
	IsDefault types.Bool   `tfsdk:"is_default"`
}

func datasourceDataSources() *common.DataSource {
	return common.NewDataSource(common.CategoryGrafanaOSS, "grafana_data_sources", &dataSourcesDataSource{})
}

func (d *dataSourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "grafana_data_sources"
}

func (d *dataSourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all data sources returned by the Grafana API for the organization, including provisioned data sources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this data source.",
			},
			"org_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Organization ID. If not set, the default organization is used for basic authentication, or the one that owns your service account for token authentication.",
			},
			"data_sources": schema.ListAttribute{
				Computed:    true,
				Description: "The organization's data sources, sorted by UID. Each entry contains the numeric `id`, unique `uid`, `name`, `type`, `url`, and `is_default` flag.",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":         types.Int64Type,
						"uid":        types.StringType,
						"name":       types.StringType,
						"type":       types.StringType,
						"url":        types.StringType,
						"is_default": types.BoolType,
					},
				},
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
		resp.Diagnostics.AddError("Failed to create Grafana client", err.Error())
		return
	}

	result, err := client.Datasources.GetDataSources()
	if err != nil {
		resp.Diagnostics.AddError("Failed to list data sources", err.Error())
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, "data_sources"))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.DataSources = make([]dataSourceListElementModel, 0, len(result.Payload))
	for _, source := range result.Payload {
		data.DataSources = append(data.DataSources, dataSourceListElementModel{
			ID:        types.Int64Value(source.ID),
			UID:       types.StringValue(source.UID),
			Name:      types.StringValue(source.Name),
			Type:      types.StringValue(source.Type),
			URL:       types.StringValue(source.URL),
			IsDefault: types.BoolValue(source.IsDefault),
		})
	}
	sort.Slice(data.DataSources, func(first, second int) bool {
		return data.DataSources[first].UID.ValueString() < data.DataSources[second].UID.ValueString()
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
