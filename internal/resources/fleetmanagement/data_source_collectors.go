package fleetmanagement

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1/collectorv1connect"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	collectorsTypeName = "grafana_fleet_management_collectors"
)

var (
	_ datasource.DataSource              = &collectorsDataSource{}
	_ datasource.DataSourceWithConfigure = &collectorsDataSource{}
)

type collectorsDataSource struct {
	client collectorv1connect.CollectorServiceClient
}

func newCollectorsDataSource() *common.DataSource {
	return common.NewDataSource(
		common.CategoryFleetManagement,
		collectorsTypeName,
		&collectorsDataSource{},
	)
}

func (d *collectorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil || d.client != nil {
		return
	}

	client, err := withClientForDataSource(req, resp)
	if err != nil {
		return
	}

	d.client = client.CollectorServiceClient
}

func (d *collectorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = collectorsTypeName
}

func (d *collectorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Represents a list of Grafana Fleet Management collectors.

* [Official documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/)
* [API documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/api-reference/collector-api/)

Required access policy scopes:

* fleet-management:read
`,
		Attributes: map[string]schema.Attribute{
			"collectors": schema.ListAttribute{
				Description: "List of collectors",
				Computed:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":                types.StringType,
						"remote_attributes": types.MapType{ElemType: types.StringType},
						"local_attributes":  types.MapType{ElemType: types.StringType},
						"enabled":           types.BoolType,
					},
				},
			},
		},
	}
}

func (d *collectorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	listReq := &collectorv1.ListCollectorsRequest{}
	listResp, err := d.client.ListCollectors(ctx, connect.NewRequest(listReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to list collectors", err.Error())
		return
	}

	collectors := make([]collectorDataSourceModel, len(listResp.Msg.Collectors))
	for i, collector := range listResp.Msg.Collectors {
		collector, diags := collectorMessageToDataSourceModel(ctx, collector)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		collectors[i] = *collector
	}

	// Sort to produce a deterministic output
	sort.Slice(collectors, func(i, j int) bool {
		return collectors[i].ID.ValueString() < collectors[j].ID.ValueString()
	})

	state := &collectorDataSourcesModel{
		Collectors: collectors,
	}
	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
