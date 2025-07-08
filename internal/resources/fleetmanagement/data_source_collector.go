package fleetmanagement

import (
	"context"

	"connectrpc.com/connect"
	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1/collectorv1connect"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &collectorDataSource{}
	_ datasource.DataSourceWithConfigure = &collectorDataSource{}
)

type collectorDataSource struct {
	client collectorv1connect.CollectorServiceClient
}

func newCollectorDataSource() *common.DataSource {
	return common.NewDataSource(
		common.CategoryFleetManagement,
		collectorTypeName,
		&collectorDataSource{},
	)
}

func (d *collectorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil || d.client != nil {
		return
	}

	client, err := withClientForDataSource(req, resp)
	if err != nil {
		return
	}

	d.client = client.CollectorServiceClient
}

func (d *collectorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = collectorTypeName
}

func (d *collectorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Represents a Grafana Fleet Management collector.

* [Official documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/)
* [API documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/api-reference/collector-api/)

Required access policy scopes:

* fleet-management:read
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the collector",
				Required:    true,
			},
			"remote_attributes": schema.MapAttribute{
				Description: "Remote attributes for the collector",
				Computed:    true,
				ElementType: types.StringType,
			},
			"local_attributes": schema.MapAttribute{
				Description: "Local attributes for the collector",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether remote configuration for the collector is enabled or not. If the collector is disabled, " +
					"it will receive empty configurations from the Fleet Management service",
				Computed: true,
			},
		},
	}
}

func (d *collectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &collectorDataSourceModel{}
	diags := req.Config.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &collectorv1.GetCollectorRequest{
		Id: data.ID.ValueString(),
	}
	getResp, err := d.client.GetCollector(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get collector", err.Error())
		return
	}

	state, diags := collectorMessageToDataSourceModel(ctx, getResp.Msg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
