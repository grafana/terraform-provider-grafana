package cloud

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CloudStackDataSource{}
var _ datasource.DataSourceWithConfigure = &CloudStackDataSource{}

var dataSourceCloudStackName = "grafana_cloud_stack"

func datasourceStack() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloud,
		dataSourceCloudStackName,
		&CloudStackDataSource{},
	)
}

type CloudStackDataSource struct {
	basePluginFrameworkDataSource
}

func (r *CloudStackDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceCloudStackName
}

func (r *CloudStackDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for Grafana Stack",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The stack id assigned to this stack by Grafana.",
			},
			"slug": schema.StringAttribute{
				Required:    true,
				Description: `Subdomain that the Grafana instance will be available at (i.e. setting slug to "<stack_slug>" will make the instance available at "https://<stack_slug>.grafana.net".`,
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of stack. Conventionally matches the url of the instance (e.g. `<stack_slug>.grafana.net`).",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of stack.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Custom URL for the Grafana instance.",
			},
			"region_slug": schema.StringAttribute{
				Computed:    true,
				Description: "The region this stack is deployed to.",
			},
			"cluster_slug": schema.StringAttribute{
				Computed:    true,
				Description: "Slug of the cluster where this stack resides.",
			},
			"cluster_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the cluster where this stack resides.",
			},
			"org_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Organization id to assign to this stack.",
			},
			"org_slug": schema.StringAttribute{
				Computed:    true,
				Description: "Organization slug to assign to this stack.",
			},
			"org_name": schema.StringAttribute{
				Computed:    true,
				Description: "Organization name to assign to this stack.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the stack.",
			},
			"labels": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "A map of labels assigned to the stack.",
			},
			"delete_protection": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether delete protection is enabled for the stack.",
			},

			// IP Allow List CNAMEs
			"grafanas_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the grafana instance (Optional)",
			},

			// Prometheus (Metrics/Mimir)
			"prometheus_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Prometheus user ID. Used for e.g. remote_write.",
			},
			"prometheus_url": schema.StringAttribute{
				Computed:    true,
				Description: "Prometheus url for this instance.",
			},
			"prometheus_name": schema.StringAttribute{
				Computed:    true,
				Description: "Prometheus name for this instance.",
			},
			"prometheus_remote_endpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Use this URL to query hosted metrics data e.g. Prometheus data source in Grafana",
			},
			"prometheus_remote_write_endpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Use this URL to send prometheus metrics to Grafana cloud",
			},
			"prometheus_status": schema.StringAttribute{
				Computed:    true,
				Description: "Prometheus status for this instance.",
			},
			"prometheus_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Prometheus when using AWS PrivateLink (only for AWS stacks)",
			},
			"prometheus_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Prometheus when using AWS PrivateLink (only for AWS stacks)",
			},
			"prometheus_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Prometheus instance (Optional)",
			},

			// Alertmanager
			"alertmanager_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "User ID of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Alertmanager instances (Optional)",
			},

			// OnCall
			"oncall_api_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the OnCall API instance configured for this stack.",
			},

			// Logs (Loki)
			"logs_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"logs_name": schema.StringAttribute{
				Computed: true,
			},
			"logs_url": schema.StringAttribute{
				Computed: true,
			},
			"logs_status": schema.StringAttribute{
				Computed: true,
			},
			"logs_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Logs when using AWS PrivateLink (only for AWS stacks)",
			},
			"logs_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Logs when using AWS PrivateLink (only for AWS stacks)",
			},
			"logs_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Logs instance (Optional)",
			},

			// Traces (Tempo)
			"traces_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"traces_name": schema.StringAttribute{
				Computed: true,
			},
			"traces_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the Traces instance configured for this stack. To use this in the Tempo data source in Grafana, append `/tempo` to the URL.",
			},
			"traces_status": schema.StringAttribute{
				Computed: true,
			},
			"traces_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Traces when using AWS PrivateLink (only for AWS stacks)",
			},
			"traces_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Traces when using AWS PrivateLink (only for AWS stacks)",
			},
			"traces_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Traces instance (Optional)",
			},

			// Profiles (Pyroscope)
			"profiles_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"profiles_name": schema.StringAttribute{
				Computed: true,
			},
			"profiles_url": schema.StringAttribute{
				Computed: true,
			},
			"profiles_status": schema.StringAttribute{
				Computed: true,
			},
			"profiles_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Profiles when using AWS PrivateLink (only for AWS stacks)",
			},
			"profiles_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Profiles when using AWS PrivateLink (only for AWS stacks)",
			},
			"profiles_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Profiles instance (Optional)",
			},

			// Graphite
			"graphite_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"graphite_name": schema.StringAttribute{
				Computed: true,
			},
			"graphite_url": schema.StringAttribute{
				Computed: true,
			},
			"graphite_status": schema.StringAttribute{
				Computed: true,
			},
			"graphite_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Graphite when using AWS PrivateLink (only for AWS stacks)",
			},
			"graphite_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Graphite when using AWS PrivateLink (only for AWS stacks)",
			},
			"graphite_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Graphite instance (Optional)",
			},

			// Fleet Management
			"fleet_management_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "User ID of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Fleet Management when using AWS PrivateLink (only for AWS stacks)",
			},
			"fleet_management_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Fleet Management when using AWS PrivateLink (only for AWS stacks)",
			},

			// Connections
			"influx_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the InfluxDB instance configured for this stack. The username is the same as the metrics' (`prometheus_user_id` attribute of this resource). See https://grafana.com/docs/grafana-cloud/send-data/metrics/metrics-influxdb/push-from-telegraf/ for docs on how to use this.",
			},
			"otlp_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the OTLP instance configured for this stack. The username is the stack's ID (`id` attribute of this resource). See https://grafana.com/docs/grafana-cloud/send-data/otlp/send-data-otlp/ for docs on how to use this.",
			},
			"otlp_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for OTLP when using AWS PrivateLink (only for AWS stacks)",
			},
			"otlp_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for OTLP when using AWS PrivateLink (only for AWS stacks)",
			},

			// PDC
			"pdc_api_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for PDC's API when using AWS PrivateLink (only for AWS stacks)",
			},
			"pdc_api_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for PDC's API when using AWS PrivateLink (only for AWS stacks)",
			},
			"pdc_gateway_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for PDC's Gateway when using AWS PrivateLink (only for AWS stacks)",
			},
			"pdc_gateway_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for PDC's Gateway when using AWS PrivateLink (only for AWS stacks)",
			},

			// Note: wait_for_readiness and wait_for_readiness_timeout are not included as they are resource-only attributes
		},
	}
}

func (r *CloudStackDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into a temporary struct with just the slug
	var data struct {
		Slug types.String `tfsdk:"slug"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the shared read function
	model, diags := ReadStackData(ctx, r.client, data.Slug.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
