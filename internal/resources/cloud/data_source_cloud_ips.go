package cloud

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CloudIPsDataSource{}
var _ datasource.DataSourceWithConfigure = &CloudIPsDataSource{}

var dataSourceCloudIPsName = "grafana_cloud_ips"

func datasourceIPs() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloud,
		dataSourceCloudIPsName,
		&CloudIPsDataSource{},
	)
}

type CloudIPsDataSource struct {
	basePluginFrameworkDataSource //nolint:unused // Embedded for consistency but Configure is overridden
}

// Configure overrides the base Configure to skip cloud API client validation.
// This data source fetches public IP lists and doesn't require authentication.
func (r *CloudIPsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// No cloud API client required - this data source only fetches public data
}

func (r *CloudIPsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceCloudIPsName
}

func (r *CloudIPsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Data source for retrieving sets of cloud IPs.

* [Official documentation](https://grafana.com/docs/grafana-cloud/reference/allow-list/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this datasource. This is an internal identifier used by the provider to track this datasource.",
			},
			"hosted_alerts": schema.SetAttribute{
				Computed:    true,
				Description: "Set of IP addresses that are used for hosted alerts.",
				ElementType: types.StringType,
			},
			"hosted_grafana": schema.SetAttribute{
				Computed:    true,
				Description: "Set of IP addresses that are used for hosted Grafana.",
				ElementType: types.StringType,
			},
			"hosted_metrics": schema.SetAttribute{
				Computed:    true,
				Description: "Set of IP addresses that are used for hosted metrics.",
				ElementType: types.StringType,
			},
			"hosted_traces": schema.SetAttribute{
				Computed:    true,
				Description: "Set of IP addresses that are used for hosted traces.",
				ElementType: types.StringType,
			},
			"hosted_logs": schema.SetAttribute{
				Computed:    true,
				Description: "Set of IP addresses that are used for hosted logs.",
				ElementType: types.StringType,
			},
			"hosted_profiles": schema.SetAttribute{
				Computed:    true,
				Description: "Set of IP addresses that are used for hosted profiles.",
				ElementType: types.StringType,
			},
			"hosted_otlp": schema.SetAttribute{
				Computed:    true,
				Description: "Set of IP addresses that are used for the OTLP Gateway.",
				ElementType: types.StringType,
			},
		},
	}
}

type CloudIPsDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	HostedAlerts   types.Set    `tfsdk:"hosted_alerts"`
	HostedGrafana  types.Set    `tfsdk:"hosted_grafana"`
	HostedMetrics  types.Set    `tfsdk:"hosted_metrics"`
	HostedTraces   types.Set    `tfsdk:"hosted_traces"`
	HostedLogs     types.Set    `tfsdk:"hosted_logs"`
	HostedProfiles types.Set    `tfsdk:"hosted_profiles"`
	HostedOTLP     types.Set    `tfsdk:"hosted_otlp"`
}

func (r *CloudIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data CloudIPsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	urlMap := map[string]string{
		"hosted_alerts":   "https://grafana.com/api/hosted-alerts/source-ips.txt",
		"hosted_grafana":  "https://grafana.com/api/hosted-grafana/source-ips.txt",
		"hosted_metrics":  "https://grafana.com/api/hosted-metrics/source-ips.txt",
		"hosted_traces":   "https://grafana.com/api/hosted-traces/source-ips.txt",
		"hosted_logs":     "https://grafana.com/api/hosted-logs/source-ips.txt",
		"hosted_profiles": "https://grafana.com/api/hosted-profiles/source-ips.txt",
		"hosted_otlp":     "https://grafana.com/api/hosted-otlp/source-ips.txt",
	}

	for attr, dataURL := range urlMap {
		// nolint: gosec
		httpResp, err := http.Get(dataURL)
		if err != nil {
			resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to query IPs", "error querying IPs for "+attr+" ("+dataURL+"): "+err.Error())}
			return
		}
		defer httpResp.Body.Close()

		b, err := io.ReadAll(httpResp.Body)
		if err != nil {
			resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to read response", "error reading response body for "+attr+" ("+dataURL+"): "+err.Error())}
			return
		}

		var ipStrings []string
		for _, ip := range strings.Split(string(b), "\n") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				ipStrings = append(ipStrings, ip)
			}
		}

		// Convert []string to types.Set
		ipSet, diags := types.SetValueFrom(ctx, types.StringType, ipStrings)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Set the appropriate field based on attr
		switch attr {
		case "hosted_alerts":
			data.HostedAlerts = ipSet
		case "hosted_grafana":
			data.HostedGrafana = ipSet
		case "hosted_metrics":
			data.HostedMetrics = ipSet
		case "hosted_traces":
			data.HostedTraces = ipSet
		case "hosted_logs":
			data.HostedLogs = ipSet
		case "hosted_profiles":
			data.HostedProfiles = ipSet
		case "hosted_otlp":
			data.HostedOTLP = ipSet
		}
	}

	data.ID = types.StringValue("cloud_ips")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
