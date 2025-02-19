package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProviderConfig struct {
	URL              types.String `tfsdk:"url"`
	Auth             types.String `tfsdk:"auth"`
	HTTPHeaders      types.Map    `tfsdk:"http_headers"`
	Retries          types.Int64  `tfsdk:"retries"`
	RetryStatusCodes types.Set    `tfsdk:"retry_status_codes"`
	RetryWait        types.Int64  `tfsdk:"retry_wait"`

	TLSKey             types.String `tfsdk:"tls_key"`
	TLSCert            types.String `tfsdk:"tls_cert"`
	CACert             types.String `tfsdk:"ca_cert"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`

	StoreDashboardSha256 types.Bool `tfsdk:"store_dashboard_sha256"`

	CloudAccessPolicyToken types.String `tfsdk:"cloud_access_policy_token"`
	CloudAPIURL            types.String `tfsdk:"cloud_api_url"`

	SMAccessToken types.String `tfsdk:"sm_access_token"`
	SMURL         types.String `tfsdk:"sm_url"`

	OncallAccessToken types.String `tfsdk:"oncall_access_token"`
	OncallURL         types.String `tfsdk:"oncall_url"`

	CloudProviderAccessToken types.String `tfsdk:"cloud_provider_access_token"`
	CloudProviderURL         types.String `tfsdk:"cloud_provider_url"`

	ConnectionsAPIAccessToken types.String `tfsdk:"connections_api_access_token"`
	ConnectionsAPIURL         types.String `tfsdk:"connections_api_url"`

	FleetManagementAuth types.String `tfsdk:"fleet_management_auth"`
	FleetManagementURL  types.String `tfsdk:"fleet_management_url"`

	UserAgent types.String `tfsdk:"-"`
	Version   types.String `tfsdk:"-"`
}

func (c *ProviderConfig) SetDefaults() error {
	var err error

	c.URL = envDefaultFuncString(c.URL, "GRAFANA_URL")
	c.Auth = envDefaultFuncString(c.Auth, "GRAFANA_AUTH")
	c.TLSKey = envDefaultFuncString(c.TLSKey, "GRAFANA_TLS_KEY")
	c.TLSCert = envDefaultFuncString(c.TLSCert, "GRAFANA_TLS_CERT")
	c.CACert = envDefaultFuncString(c.CACert, "GRAFANA_CA_CERT")
	c.CloudAccessPolicyToken = envDefaultFuncString(c.CloudAccessPolicyToken, "GRAFANA_CLOUD_ACCESS_POLICY_TOKEN")
	c.CloudAPIURL = envDefaultFuncString(c.CloudAPIURL, "GRAFANA_CLOUD_API_URL", "https://grafana.com")
	c.SMAccessToken = envDefaultFuncString(c.SMAccessToken, "GRAFANA_SM_ACCESS_TOKEN")
	c.SMURL = envDefaultFuncString(c.SMURL, "GRAFANA_SM_URL", "https://synthetic-monitoring-api.grafana.net")
	c.OncallAccessToken = envDefaultFuncString(c.OncallAccessToken, "GRAFANA_ONCALL_ACCESS_TOKEN")
	c.OncallURL = envDefaultFuncString(c.OncallURL, "GRAFANA_ONCALL_URL", "https://oncall-prod-us-central-0.grafana.net/oncall")
	c.CloudProviderAccessToken = envDefaultFuncString(c.CloudProviderAccessToken, "GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN")
	c.CloudProviderURL = envDefaultFuncString(c.CloudProviderURL, "GRAFANA_CLOUD_PROVIDER_URL")
	c.ConnectionsAPIAccessToken = envDefaultFuncString(c.ConnectionsAPIAccessToken, "GRAFANA_CONNECTIONS_API_ACCESS_TOKEN")
	c.ConnectionsAPIURL = envDefaultFuncString(c.ConnectionsAPIURL, "GRAFANA_CONNECTIONS_API_URL", "https://connections-api.grafana.net")
	c.FleetManagementAuth = envDefaultFuncString(c.FleetManagementAuth, "GRAFANA_FLEET_MANAGEMENT_AUTH")
	c.FleetManagementURL = envDefaultFuncString(c.FleetManagementURL, "GRAFANA_FLEET_MANAGEMENT_URL")
	if c.StoreDashboardSha256, err = envDefaultFuncBool(c.StoreDashboardSha256, "GRAFANA_STORE_DASHBOARD_SHA256", false); err != nil {
		return fmt.Errorf("failed to parse GRAFANA_STORE_DASHBOARD_SHA256: %w", err)
	}
	if c.Retries, err = envDefaultFuncInt64(c.Retries, "GRAFANA_RETRIES", 3); err != nil {
		return fmt.Errorf("failed to parse GRAFANA_RETRIES: %w", err)
	}
	if c.RetryWait, err = envDefaultFuncInt64(c.RetryWait, "GRAFANA_RETRY_WAIT", 0); err != nil {
		return fmt.Errorf("failed to parse GRAFANA_RETRY_WAIT: %w", err)
	}
	if c.InsecureSkipVerify, err = envDefaultFuncBool(c.InsecureSkipVerify, "GRAFANA_INSECURE_SKIP_VERIFY", false); err != nil {
		return fmt.Errorf("failed to parse GRAFANA_INSECURE_SKIP_VERIFY: %w", err)
	}

	if envValue := os.Getenv("GRAFANA_HTTP_HEADERS"); c.HTTPHeaders.IsNull() && envValue != "" {
		headersMap := make(map[string]string)
		if err := json.Unmarshal([]byte(envValue), &headersMap); err != nil {
			return fmt.Errorf("failed to parse GRAFANA_HTTP_HEADERS: %w", err)
		}
		headersValue := map[string]attr.Value{}
		for k, v := range headersMap {
			headersValue[k] = types.StringValue(v)
		}
		c.HTTPHeaders = types.MapValueMust(types.StringType, headersValue)
	}

	if envValue := os.Getenv("GRAFANA_RETRY_STATUS_CODES"); c.RetryStatusCodes.IsNull() && envValue != "" {
		retryStatusCodes := []attr.Value{}
		for _, code := range strings.Split(envValue, ",") {
			retryStatusCodes = append(retryStatusCodes, types.StringValue(code))
		}
		c.RetryStatusCodes = types.SetValueMust(types.StringType, retryStatusCodes)
	} else if c.RetryStatusCodes.IsNull() {
		c.RetryStatusCodes = types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("429"),
			types.StringValue("5xx"),
			types.StringValue("401"), // In high load scenarios, Grafana sometimes returns 401s (unable to authenticate the user?)
		})
	}

	return nil
}

type frameworkProvider struct {
	version string
}

func (p *frameworkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "grafana"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *frameworkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The root URL of a Grafana server. May alternatively be set via the `GRAFANA_URL` environment variable.",
			},
			"auth": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API token, basic auth in the `username:password` format or `anonymous` (string literal). May alternatively be set via the `GRAFANA_AUTH` environment variable.",
			},
			"http_headers": schema.MapAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional. HTTP headers mapping keys to values used for accessing the Grafana and Grafana Cloud APIs. May alternatively be set via the `GRAFANA_HTTP_HEADERS` environment variable in JSON format.",
				ElementType:         types.StringType,
			},
			"retries": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The amount of retries to use for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRIES` environment variable.",
			},
			"retry_status_codes": schema.SetAttribute{
				Optional:            true,
				MarkdownDescription: "The status codes to retry on for Grafana API and Grafana Cloud API calls. Use `x` as a digit wildcard. Defaults to 429 and 5xx. May alternatively be set via the `GRAFANA_RETRY_STATUS_CODES` environment variable.",
				ElementType:         types.StringType,
			},
			"retry_wait": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The amount of time in seconds to wait between retries for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRY_WAIT` environment variable.",
			},
			"tls_key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Client TLS key (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_KEY` environment variable.",
			},
			"tls_cert": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Client TLS certificate (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_CERT` environment variable.",
			},
			"ca_cert": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Certificate CA bundle (file path or literal value) to use to verify the Grafana server's certificate. May alternatively be set via the `GRAFANA_CA_CERT` environment variable.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Skip TLS certificate verification. May alternatively be set via the `GRAFANA_INSECURE_SKIP_VERIFY` environment variable.",
			},
			"store_dashboard_sha256": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to true if you want to save only the sha256sum instead of complete dashboard model JSON in the tfstate.",
			},

			"cloud_access_policy_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Access Policy Token for Grafana Cloud. May alternatively be set via the `GRAFANA_CLOUD_ACCESS_POLICY_TOKEN` environment variable.",
			},
			"cloud_api_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Grafana Cloud's API URL. May alternatively be set via the `GRAFANA_CLOUD_API_URL` environment variable.",
			},

			"sm_access_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.",
			},
			"sm_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/set-up/set-up-private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.",
			},

			"oncall_access_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "A Grafana OnCall access token. May alternatively be set via the `GRAFANA_ONCALL_ACCESS_TOKEN` environment variable.",
			},
			"oncall_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "An Grafana OnCall backend address. May alternatively be set via the `GRAFANA_ONCALL_URL` environment variable.",
			},

			"cloud_provider_access_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "A Grafana Cloud Provider access token. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN` environment variable.",
			},
			"cloud_provider_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A Grafana Cloud Provider backend address. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_URL` environment variable.",
			},

			"connections_api_access_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "A Grafana Connections API access token. May alternatively be set via the `GRAFANA_CONNECTIONS_API_ACCESS_TOKEN` environment variable.",
			},
			"connections_api_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A Grafana Connections API address. May alternatively be set via the `GRAFANA_CONNECTIONS_API_URL` environment variable.",
			},

			"fleet_management_auth": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "A Grafana Fleet Management basic auth in the `username:password` format. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_AUTH` environment variable.",
			},
			"fleet_management_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A Grafana Fleet Management API address. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_URL` environment variable.",
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *frameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg ProviderConfig
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := cfg.SetDefaults(); err != nil {
		resp.Diagnostics.AddError("failed to set defaults", err.Error())
		return
	}
	cfg.Version = types.StringValue(p.version)
	cfg.UserAgent = types.StringValue(fmt.Sprintf("Terraform/%s (+https://www.terraform.io) terraform-provider-grafana/%s", req.TerraformVersion, p.version))

	clients, err := CreateClients(cfg)
	if err != nil {
		resp.Diagnostics.AddError("failed to create clients", err.Error())
		return
	}

	resp.ResourceData = clients
	resp.DataSourceData = clients
}

// DataSources defines the data sources implemented in the provider.
func (p *frameworkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return pluginFrameworkDataSources()
}

// Resources defines the resources implemented in the provider.
func (p *frameworkProvider) Resources(_ context.Context) []func() resource.Resource {
	return pluginFrameworkResources()
}

// FrameworkProvider returns a terraform-plugin-framework Provider.
// This is the recommended way forward for new resources.
func FrameworkProvider(version string) provider.Provider {
	return &frameworkProvider{
		version: version,
	}
}

func envDefaultFuncString(v types.String, envVar string, defaultValue ...string) types.String {
	if envValue := os.Getenv(envVar); v.IsNull() && envValue != "" {
		return types.StringValue(envValue)
	} else if v.IsNull() && len(defaultValue) > 0 {
		return types.StringValue(defaultValue[0])
	}
	return v
}

func envDefaultFuncInt64(v types.Int64, envVar string, defaultValue ...int64) (types.Int64, error) {
	if envValue := os.Getenv(envVar); v.IsNull() && envValue != "" {
		value, err := strconv.ParseInt(envValue, 10, 64)
		return types.Int64Value(value), err
	} else if v.IsNull() && len(defaultValue) > 0 {
		return types.Int64Value(defaultValue[0]), nil
	}
	return v, nil
}

func envDefaultFuncBool(v types.Bool, envVar string, defaultValue ...bool) (types.Bool, error) {
	if envValue := os.Getenv(envVar); v.IsNull() && envValue != "" {
		value, err := strconv.ParseBool(envValue)
		return types.BoolValue(value), err
	} else if v.IsNull() && len(defaultValue) > 0 {
		return types.BoolValue(defaultValue[0]), nil
	}
	return v, nil
}
