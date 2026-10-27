package syntheticmonitoring

import (
	"context"
	"fmt"

	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*probesDataSource)(nil)

func datasourceProbes() *common.DataSource {
	return common.NewDataSource(
		common.CategorySyntheticMonitoring,
		"grafana_synthetic_monitoring_probes",
		&probesDataSource{},
	)
}

type probesDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	FilterDeprecated types.Bool   `tfsdk:"filter_deprecated"`
	Probes           types.Map    `tfsdk:"probes"`
}

type probesDataSource struct {
	client *smapi.Client
}

func (d *probesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil || d.client != nil {
		return
	}
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	if client.SMAPI == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is not configured for Synthetic Monitoring.",
			"Please ensure that sm_access_token is set in the provider configuration.",
		)
		return
	}
	d.client = client.SMAPI
}

func (d *probesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "grafana_synthetic_monitoring_probes"
}

func (d *probesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for retrieving all probes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"filter_deprecated": schema.BoolAttribute{
				Optional:    true,
				Description: "If true, only probes that are not deprecated will be returned. Defaults to `true`.",
			},
			"probes": schema.MapAttribute{
				ElementType: types.Int64Type,
				Computed:    true,
				Description: "Map of probes with their names as keys and IDs as values.",
			},
		},
	}
}

func (d *probesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data probesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prbs, err := d.client.ListProbes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list probes", err.Error())
		return
	}

	filterDeprecated := data.FilterDeprecated.IsNull() || data.FilterDeprecated.ValueBool()
	probesMap := make(map[string]int64, len(prbs))
	for _, p := range prbs {
		if !p.Deprecated || !filterDeprecated {
			probesMap[p.Name] = p.Id
		}
	}

	probesVal, diags := types.MapValueFrom(ctx, types.Int64Type, probesMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue("probes")
	data.Probes = probesVal
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
