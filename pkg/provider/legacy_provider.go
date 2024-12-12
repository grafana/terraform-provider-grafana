package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	EnableGenerateEnvVar     = "TF_GENERATE_UNSENSITIVE"
	EnableGenerateMarkerFile = ".generate-make-all-fields-unsensitive"
)

func init() {
	schema.DescriptionKind = schema.StringMarkdown
	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		return strings.TrimSpace(desc)
	}
}

// Provider returns a terraform-provider-sdk2 provider.
// This is the deprecated way of creating a provider, and should only be used for legacy resources.
func Provider(version string) *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The root URL of a Grafana server. May alternatively be set via the `GRAFANA_URL` environment variable.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"auth": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "API token, basic auth in the `username:password` format or `anonymous` (string literal). May alternatively be set via the `GRAFANA_AUTH` environment variable.",
			},
			"http_headers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Optional. HTTP headers mapping keys to values used for accessing the Grafana and Grafana Cloud APIs. May alternatively be set via the `GRAFANA_HTTP_HEADERS` environment variable in JSON format.",
			},
			"retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The amount of retries to use for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRIES` environment variable.",
			},
			"retry_status_codes": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The status codes to retry on for Grafana API and Grafana Cloud API calls. Use `x` as a digit wildcard. Defaults to 429 and 5xx. May alternatively be set via the `GRAFANA_RETRY_STATUS_CODES` environment variable.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"retry_wait": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The amount of time in seconds to wait between retries for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRY_WAIT` environment variable.",
			},
			"tls_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Client TLS key (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_KEY` environment variable.",
			},
			"tls_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Client TLS certificate (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_CERT` environment variable.",
			},
			"ca_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Certificate CA bundle (file path or literal value) to use to verify the Grafana server's certificate. May alternatively be set via the `GRAFANA_CA_CERT` environment variable.",
			},
			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip TLS certificate verification. May alternatively be set via the `GRAFANA_INSECURE_SKIP_VERIFY` environment variable.",
			},

			"cloud_access_policy_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Access Policy Token for Grafana Cloud. May alternatively be set via the `GRAFANA_CLOUD_ACCESS_POLICY_TOKEN` environment variable.",
			},
			"cloud_api_url": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Grafana Cloud's API URL. May alternatively be set via the `GRAFANA_CLOUD_API_URL` environment variable.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},

			"sm_access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.",
			},
			"sm_url": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/set-up/set-up-private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"store_dashboard_sha256": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to true if you want to save only the sha256sum instead of complete dashboard model JSON in the tfstate.",
			},

			"oncall_access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "A Grafana OnCall access token. May alternatively be set via the `GRAFANA_ONCALL_ACCESS_TOKEN` environment variable.",
			},
			"oncall_url": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "An Grafana OnCall backend address. May alternatively be set via the `GRAFANA_ONCALL_URL` environment variable.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},

			"cloud_provider_access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "A Grafana Cloud Provider access token. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN` environment variable.",
			},
			"cloud_provider_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A Grafana Cloud Provider backend address. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_URL` environment variable.",
			},

			"connections_api_access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "A Grafana Connections API access token. May alternatively be set via the `GRAFANA_CONNECTIONS_API_ACCESS_TOKEN` environment variable.",
			},
			"connections_api_url": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "A Grafana Connections API address. May alternatively be set via the `GRAFANA_CONNECTIONS_API_URL` environment variable.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},

			"fleet_management_auth": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "A Grafana Fleet Management basic auth in the `username:password` format. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_AUTH` environment variable.",
			},
			"fleet_management_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A Grafana Fleet Management API address. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_URL` environment variable.",
			},
		},

		ResourcesMap:   legacySDKResources(),
		DataSourcesMap: legacySDKDataSources(),
	}

	if os.Getenv(EnableGenerateEnvVar) != "" {
		// If TF_GENERATE_UNSENSITIVE envvar is set and there's the "marker file" in the current directory,
		// generate the provider with all fields marked as non-sensitive.
		// The Terraform generation feature is overly-aggressive in redacting sensitive fields, it redacts all blocks at the root level.
		// Security note:
		// Setting an envvar + creating a marker file in the TF dir means that the user has full control over the TF context.
		// This means that the user could also read sensitive data from the state, or use the `unsensitive` TF function to read sensitive data.
		// So, this feature doesn't introduce a new way to extract sensitive data.

		wd, err := os.Getwd()
		if err != nil {
			panic(err) // It's ok to panic, this is only meant to be used in the context of the generator.
		}
		_, err = os.Stat(filepath.Join(wd, EnableGenerateMarkerFile))
		if err == nil {
			for k := range p.ResourcesMap {
				unsensitive(p.ResourcesMap[k])
			}
		} else {
			fmt.Println("The marker file for generating unsensitive fields is not present, skipping.")
		}
	}

	p.ConfigureContextFunc = configure(version, p)

	return p
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		// Convert SDK config to "plugin-framework" format
		headers := types.MapNull(types.StringType)
		if v, ok := d.GetOk("http_headers"); ok {
			headersValue := map[string]attr.Value{}
			for k, v := range v.(map[string]interface{}) {
				headersValue[k] = types.StringValue(v.(string))
			}
			headers = types.MapValueMust(types.StringType, headersValue)
		}

		statusCodes := types.SetNull(types.StringType)
		if v, ok := d.GetOk("retry_status_codes"); ok {
			statusCodesValue := []attr.Value{}
			for _, v := range v.(*schema.Set).List() {
				statusCodesValue = append(statusCodesValue, types.StringValue(v.(string)))
			}
			statusCodes = types.SetValueMust(types.StringType, statusCodesValue)
		}

		cfg := ProviderConfig{
			Auth:                      stringValueOrNull(d, "auth"),
			URL:                       stringValueOrNull(d, "url"),
			TLSKey:                    stringValueOrNull(d, "tls_key"),
			TLSCert:                   stringValueOrNull(d, "tls_cert"),
			CACert:                    stringValueOrNull(d, "ca_cert"),
			InsecureSkipVerify:        boolValueOrNull(d, "insecure_skip_verify"),
			CloudAccessPolicyToken:    stringValueOrNull(d, "cloud_access_policy_token"),
			CloudAPIURL:               stringValueOrNull(d, "cloud_api_url"),
			SMAccessToken:             stringValueOrNull(d, "sm_access_token"),
			SMURL:                     stringValueOrNull(d, "sm_url"),
			OncallAccessToken:         stringValueOrNull(d, "oncall_access_token"),
			OncallURL:                 stringValueOrNull(d, "oncall_url"),
			CloudProviderAccessToken:  stringValueOrNull(d, "cloud_provider_access_token"),
			CloudProviderURL:          stringValueOrNull(d, "cloud_provider_url"),
			ConnectionsAPIAccessToken: stringValueOrNull(d, "connections_api_access_token"),
			ConnectionsAPIURL:         stringValueOrNull(d, "connections_api_url"),
			FleetManagementAuth:       stringValueOrNull(d, "fleet_management_auth"),
			FleetManagementURL:        stringValueOrNull(d, "fleet_management_url"),
			StoreDashboardSha256:      boolValueOrNull(d, "store_dashboard_sha256"),
			HTTPHeaders:               headers,
			Retries:                   int64ValueOrNull(d, "retries"),
			RetryStatusCodes:          statusCodes,
			RetryWait:                 types.Int64Value(int64(d.Get("retry_wait").(int))),
			UserAgent:                 types.StringValue(p.UserAgent("terraform-provider-grafana", version)),
			Version:                   types.StringValue(version),
		}
		if err := cfg.SetDefaults(); err != nil {
			return nil, diag.FromErr(err)
		}

		clients, err := CreateClients(cfg)
		return clients, diag.FromErr(err)
	}
}

func stringValueOrNull(d *schema.ResourceData, key string) types.String {
	if v, ok := d.GetOk(key); ok {
		return types.StringValue(v.(string))
	}
	return types.StringNull()
}

func boolValueOrNull(d *schema.ResourceData, key string) types.Bool {
	if v, ok := d.GetOk(key); ok {
		return types.BoolValue(v.(bool))
	}
	return types.BoolNull()
}

func int64ValueOrNull(d *schema.ResourceData, key string) types.Int64 {
	if v, ok := d.GetOk(key); ok {
		return types.Int64Value(int64(v.(int)))
	}
	return types.Int64Null()
}

func unsensitive(r *schema.Resource) {
	for _, s := range r.Schema {
		s.Sensitive = false
		if r, ok := s.Elem.(*schema.Resource); ok {
			unsensitive(r)
		}
		if _, ok := s.Elem.(*schema.Schema); ok {
			s.Elem.(*schema.Schema).Sensitive = false
		}
	}
}
