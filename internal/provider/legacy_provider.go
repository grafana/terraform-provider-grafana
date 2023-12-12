package provider

import (
	"context"

	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/internal/resources/oncall"
	"github.com/grafana/terraform-provider-grafana/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/internal/resources/syntheticmonitoring"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	var (
		// Resources that require the Grafana client to exist.
		grafanaClientResources = addResourcesMetadataValidation(grafanaClientPresent, map[string]*schema.Resource{
			// Grafana
			"grafana_annotation":                 grafana.ResourceAnnotation(),
			"grafana_api_key":                    grafana.ResourceAPIKey(),
			"grafana_contact_point":              grafana.ResourceContactPoint(),
			"grafana_dashboard":                  grafana.ResourceDashboard(),
			"grafana_dashboard_public":           grafana.ResourcePublicDashboard(),
			"grafana_dashboard_permission":       grafana.ResourceDashboardPermission(),
			"grafana_data_source":                grafana.ResourceDataSource(),
			"grafana_data_source_permission":     grafana.ResourceDatasourcePermission(),
			"grafana_folder":                     grafana.ResourceFolder(),
			"grafana_folder_permission":          grafana.ResourceFolderPermission(),
			"grafana_library_panel":              grafana.ResourceLibraryPanel(),
			"grafana_message_template":           grafana.ResourceMessageTemplate(),
			"grafana_mute_timing":                grafana.ResourceMuteTiming(),
			"grafana_notification_policy":        grafana.ResourceNotificationPolicy(),
			"grafana_organization":               grafana.ResourceOrganization(),
			"grafana_organization_preferences":   grafana.ResourceOrganizationPreferences(),
			"grafana_playlist":                   grafana.ResourcePlaylist(),
			"grafana_report":                     grafana.ResourceReport(),
			"grafana_role":                       grafana.ResourceRole(),
			"grafana_role_assignment":            grafana.ResourceRoleAssignment(),
			"grafana_rule_group":                 grafana.ResourceRuleGroup(),
			"grafana_team":                       grafana.ResourceTeam(),
			"grafana_team_external_group":        grafana.ResourceTeamExternalGroup(),
			"grafana_service_account_token":      grafana.ResourceServiceAccountToken(),
			"grafana_service_account":            grafana.ResourceServiceAccount(),
			"grafana_service_account_permission": grafana.ResourceServiceAccountPermission(),
			"grafana_user":                       grafana.ResourceUser(),

			// Machine Learning
			"grafana_machine_learning_job":              machinelearning.ResourceJob(),
			"grafana_machine_learning_holiday":          machinelearning.ResourceHoliday(),
			"grafana_machine_learning_outlier_detector": machinelearning.ResourceOutlierDetector(),

			// SLO
			"grafana_slo": slo.ResourceSlo(),
		})

		// Resources that require the Synthetic Monitoring client to exist.
		smClientResources = addResourcesMetadataValidation(smClientPresent, map[string]*schema.Resource{
			"grafana_synthetic_monitoring_check": syntheticmonitoring.ResourceCheck(),
			"grafana_synthetic_monitoring_probe": syntheticmonitoring.ResourceProbe(),
		})

		// Resources that require the Cloud client to exist.
		cloudClientResources = addResourcesMetadataValidation(cloudClientPresent, map[string]*schema.Resource{
			"grafana_cloud_access_policy":               cloud.ResourceAccessPolicy(),
			"grafana_cloud_access_policy_token":         cloud.ResourceAccessPolicyToken(),
			"grafana_cloud_api_key":                     cloud.ResourceAPIKey(),
			"grafana_cloud_plugin_installation":         cloud.ResourcePluginInstallation(),
			"grafana_cloud_stack":                       cloud.ResourceStack(),
			"grafana_cloud_stack_api_key":               cloud.ResourceStackAPIKey(),
			"grafana_cloud_stack_service_account":       cloud.ResourceStackServiceAccount(),
			"grafana_cloud_stack_service_account_token": cloud.ResourceStackServiceAccountToken(),
		})

		// Resources that require the OnCall client to exist.
		onCallClientResources = addResourcesMetadataValidation(onCallClientPresent, map[string]*schema.Resource{
			"grafana_oncall_integration":      oncall.ResourceIntegration(),
			"grafana_oncall_route":            oncall.ResourceRoute(),
			"grafana_oncall_escalation_chain": oncall.ResourceEscalationChain(),
			"grafana_oncall_escalation":       oncall.ResourceEscalation(),
			"grafana_oncall_on_call_shift":    oncall.ResourceOnCallShift(),
			"grafana_oncall_schedule":         oncall.ResourceSchedule(),
			"grafana_oncall_outgoing_webhook": oncall.ResourceOutgoingWebhook(),
		})

		// Datasources that require the Grafana client to exist.
		grafanaClientDatasources = addResourcesMetadataValidation(grafanaClientPresent, map[string]*schema.Resource{
			"grafana_dashboard":                grafana.DatasourceDashboard(),
			"grafana_dashboards":               grafana.DatasourceDashboards(),
			"grafana_data_source":              grafana.DatasourceDatasource(),
			"grafana_folder":                   grafana.DatasourceFolder(),
			"grafana_folders":                  grafana.DatasourceFolders(),
			"grafana_library_panel":            grafana.DatasourceLibraryPanel(),
			"grafana_user":                     grafana.DatasourceUser(),
			"grafana_users":                    grafana.DatasourceUsers(),
			"grafana_role":                     grafana.DatasourceRole(),
			"grafana_team":                     grafana.DatasourceTeam(),
			"grafana_organization":             grafana.DatasourceOrganization(),
			"grafana_organization_preferences": grafana.DatasourceOrganizationPreferences(),

			// SLO
			"grafana_slos": slo.DatasourceSlo(),
		})

		// Datasources that require the Synthetic Monitoring client to exist.
		smClientDatasources = addResourcesMetadataValidation(smClientPresent, map[string]*schema.Resource{
			"grafana_synthetic_monitoring_probe":  syntheticmonitoring.DataSourceProbe(),
			"grafana_synthetic_monitoring_probes": syntheticmonitoring.DataSourceProbes(),
		})

		// Datasources that require the Cloud client to exist.
		cloudClientDatasources = addResourcesMetadataValidation(cloudClientPresent, map[string]*schema.Resource{
			"grafana_cloud_ips":          cloud.DataSourceIPs(),
			"grafana_cloud_organization": cloud.DataSourceOrganization(),
			"grafana_cloud_stack":        cloud.DataSourceStack(),
		})

		// Datasources that require the OnCall client to exist.
		onCallClientDatasources = addResourcesMetadataValidation(onCallClientPresent, map[string]*schema.Resource{
			"grafana_oncall_user":             oncall.DataSourceUser(),
			"grafana_oncall_escalation_chain": oncall.DataSourceEscalationChain(),
			"grafana_oncall_schedule":         oncall.DataSourceSchedule(),
			"grafana_oncall_slack_channel":    oncall.DataSourceSlackChannel(),
			"grafana_oncall_action":           oncall.DataSourceAction(), // deprecated
			"grafana_oncall_outgoing_webhook": oncall.DataSourceOutgoingWebhook(),
			"grafana_oncall_user_group":       oncall.DataSourceUserGroup(),
			"grafana_oncall_team":             oncall.DataSourceTeam(),
		})
	)

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
			"org_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Deprecated:  "Use the `org_id` attributes on resources instead.",
				Description: "Deprecated: Use the `org_id` attributes on resources instead.",
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

			"cloud_api_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Access Policy Token (or API key) for Grafana Cloud. May alternatively be set via the `GRAFANA_CLOUD_API_KEY` environment variable.",
			},
			"cloud_api_url": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("GRAFANA_CLOUD_API_URL", "https://grafana.com"),
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
				Description:  "Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/monitor-public-endpoints/private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.",
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
		},

		ResourcesMap: mergeResourceMaps(
			map[string]*schema.Resource{
				// This one installs SM on a cloud instance, everything it needs is in its attributes
				"grafana_synthetic_monitoring_installation": cloud.ResourceInstallation(),
			},
			grafanaClientResources,
			smClientResources,
			onCallClientResources,
			cloudClientResources,
		),

		DataSourcesMap: mergeResourceMaps(
			grafanaClientDatasources,
			smClientDatasources,
			onCallClientDatasources,
			cloudClientDatasources,
		),
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

		cfg := frameworkProviderConfig{
			Auth:                 stringValueOrNull(d, "auth"),
			URL:                  stringValueOrNull(d, "url"),
			OrgID:                int64ValueOrNull(d, "org_id"),
			TLSKey:               stringValueOrNull(d, "tls_key"),
			TLSCert:              stringValueOrNull(d, "tls_cert"),
			CACert:               stringValueOrNull(d, "ca_cert"),
			InsecureSkipVerify:   boolValueOrNull(d, "insecure_skip_verify"),
			CloudAPIKey:          stringValueOrNull(d, "cloud_api_key"),
			CloudAPIURL:          stringValueOrNull(d, "cloud_api_url"),
			SMAccessToken:        stringValueOrNull(d, "sm_access_token"),
			SMURL:                stringValueOrNull(d, "sm_url"),
			OncallAccessToken:    stringValueOrNull(d, "oncall_access_token"),
			OncallURL:            stringValueOrNull(d, "oncall_url"),
			StoreDashboardSha256: boolValueOrNull(d, "store_dashboard_sha256"),
			HTTPHeaders:          headers,
			Retries:              int64ValueOrNull(d, "retries"),
			RetryStatusCodes:     statusCodes,
			RetryWait:            types.Int64Value(int64(d.Get("retry_wait").(int))),
			UserAgent:            types.StringValue(p.UserAgent("terraform-provider-grafana", version)),
		}
		if err := cfg.SetDefaults(); err != nil {
			return nil, diag.FromErr(err)
		}

		clients, err := createClients(cfg)
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

func mergeResourceMaps(maps ...map[string]*schema.Resource) map[string]*schema.Resource {
	result := make(map[string]*schema.Resource)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
