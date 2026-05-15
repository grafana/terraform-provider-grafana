package syntheticmonitoring

import (
	"context"
	"fmt"
	"strconv"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*probeDataSource)(nil)

func datasourceProbe() *common.DataSource {
	return common.NewDataSource(
		common.CategorySyntheticMonitoring,
		"grafana_synthetic_monitoring_probe",
		&probeDataSource{},
	)
}

type probeDataSourceModel struct {
	ID                    types.String  `tfsdk:"id"`
	TenantID              types.Int64   `tfsdk:"tenant_id"`
	Name                  types.String  `tfsdk:"name"`
	Latitude              types.Float64 `tfsdk:"latitude"`
	Longitude             types.Float64 `tfsdk:"longitude"`
	Region                types.String  `tfsdk:"region"`
	Public                types.Bool    `tfsdk:"public"`
	Labels                types.Map     `tfsdk:"labels"`
	DisableScriptedChecks types.Bool    `tfsdk:"disable_scripted_checks"`
	DisableBrowserChecks  types.Bool    `tfsdk:"disable_browser_checks"`
}

type probeDataSource struct {
	client *smapi.Client
}

func (d *probeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *probeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "grafana_synthetic_monitoring_probe"
}

func (d *probeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for retrieving a single probe by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the probe.",
			},
			"tenant_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The tenant ID of the probe.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the probe.",
			},
			"latitude": schema.Float64Attribute{
				Computed:    true,
				Description: "Latitude coordinates.",
			},
			"longitude": schema.Float64Attribute{
				Computed:    true,
				Description: "Longitude coordinates.",
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "Region of the probe.",
			},
			"public": schema.BoolAttribute{
				Computed:    true,
				Description: "Public probes are run by Grafana Labs and can be used by all users. Only Grafana Labs managed public probes will be set to `true`.",
			},
			"labels": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Custom labels to be included with collected metrics and logs.",
			},
			"disable_scripted_checks": schema.BoolAttribute{
				Computed:    true,
				Description: "Disables scripted checks for this probe.",
			},
			"disable_browser_checks": schema.BoolAttribute{
				Computed:    true,
				Description: "Disables browser checks for this probe.",
			},
		},
	}
}

func (d *probeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data probeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prbs, err := d.client.ListProbes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list probes", err.Error())
		return
	}

	var prb sm.Probe
	for _, p := range prbs {
		if p.Name == data.Name.ValueString() {
			prb = p
			break
		}
	}
	if prb.Id == 0 {
		resp.Diagnostics.AddError("Probe not found", fmt.Sprintf("probe with name %q not found", data.Name.ValueString()))
		return
	}

	labels := make(map[string]string, len(prb.Labels))
	for _, l := range prb.Labels {
		labels[l.Name] = l.Value
	}
	labelsVal, diags := types.MapValueFrom(ctx, types.StringType, labels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(prb.Id, 10))
	data.TenantID = types.Int64Value(prb.TenantId)
	data.Name = types.StringValue(prb.Name)
	data.Latitude = types.Float64Value(float64(prb.Latitude))
	data.Longitude = types.Float64Value(float64(prb.Longitude))
	data.Region = types.StringValue(prb.Region)
	data.Public = types.BoolValue(prb.Public)
	data.Labels = labelsVal

	if prb.Capabilities != nil {
		data.DisableScriptedChecks = types.BoolValue(prb.Capabilities.DisableScriptedChecks)
		data.DisableBrowserChecks = types.BoolValue(prb.Capabilities.DisableBrowserChecks)
	} else {
		data.DisableScriptedChecks = types.BoolValue(false)
		data.DisableBrowserChecks = types.BoolValue(false)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
