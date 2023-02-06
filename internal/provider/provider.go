package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/machine-learning-go-client/mlapi"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/grafana"
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

func Provider(version string) func() *schema.Provider {
	var (
		// Resources that require the Grafana client to exist.
		grafanaClientResources = addResourcesMetadataValidation(grafanaClientPresent, map[string]*schema.Resource{
			// Grafana
			"grafana_annotation":                 grafana.ResourceAnnotation(),
			"grafana_alert_notification":         grafana.ResourceAlertNotification(),
			"grafana_builtin_role_assignment":    grafana.ResourceBuiltInRoleAssignment(),
			"grafana_contact_point":              grafana.ResourceContactPoint(),
			"grafana_dashboard":                  grafana.ResourceDashboard(),
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
			"grafana_team_preferences":           grafana.ResourceTeamPreferences(),
			"grafana_team_external_group":        grafana.ResourceTeamExternalGroup(),
			"grafana_service_account_token":      grafana.ResourceServiceAccountToken(),
			"grafana_service_account":            grafana.ResourceServiceAccount(),
			"grafana_service_account_permission": grafana.ResourceServiceAccountPermission(),
			"grafana_user":                       grafana.ResourceUser(),

			// Machine Learning
			"grafana_machine_learning_job":              grafana.ResourceMachineLearningJob(),
			"grafana_machine_learning_holiday":          grafana.ResourceMachineLearningHoliday(),
			"grafana_machine_learning_outlier_detector": grafana.ResourceMachineLearningOutlierDetector(),
		})

		// Resources that require the Synthetic Monitoring client to exist.
		smClientResources = addResourcesMetadataValidation(smClientPresent, map[string]*schema.Resource{
			"grafana_synthetic_monitoring_check": grafana.ResourceSyntheticMonitoringCheck(),
			"grafana_synthetic_monitoring_probe": grafana.ResourceSyntheticMonitoringProbe(),
		})

		// Resources that require the Cloud client to exist.
		cloudClientResources = addResourcesMetadataValidation(cloudClientPresent, map[string]*schema.Resource{
			"grafana_cloud_access_policy":       grafana.ResourceCloudAccessPolicy(),
			"grafana_cloud_access_policy_token": grafana.ResourceCloudAccessPolicyToken(),
			"grafana_cloud_api_key":             grafana.ResourceCloudAPIKey(),
			"grafana_cloud_plugin_installation": grafana.ResourceCloudPluginInstallation(),
			"grafana_cloud_stack":               grafana.ResourceCloudStack(),
		})

		// Resources that require the OnCall client to exist.
		onCallClientResources = addResourcesMetadataValidation(onCallClientPresent, map[string]*schema.Resource{
			"grafana_oncall_integration":      grafana.ResourceOnCallIntegration(),
			"grafana_oncall_route":            grafana.ResourceOnCallRoute(),
			"grafana_oncall_escalation_chain": grafana.ResourceOnCallEscalationChain(),
			"grafana_oncall_escalation":       grafana.ResourceOnCallEscalation(),
			"grafana_oncall_on_call_shift":    grafana.ResourceOnCallOnCallShift(),
			"grafana_oncall_schedule":         grafana.ResourceOnCallSchedule(),
			"grafana_oncall_outgoing_webhook": grafana.ResourceOnCallOutgoingWebhook(),
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
			"grafana_team":                     grafana.DatasourceTeam(),
			"grafana_organization":             grafana.DatasourceOrganization(),
			"grafana_organization_preferences": grafana.DatasourceOrganizationPreferences(),
		})

		// Datasources that require the Synthetic Monitoring client to exist.
		smClientDatasources = addResourcesMetadataValidation(smClientPresent, map[string]*schema.Resource{
			"grafana_synthetic_monitoring_probe":  grafana.DatasourceSyntheticMonitoringProbe(),
			"grafana_synthetic_monitoring_probes": grafana.DatasourceSyntheticMonitoringProbes(),
		})

		// Datasources that require the Cloud client to exist.
		cloudClientDatasources = addResourcesMetadataValidation(cloudClientPresent, map[string]*schema.Resource{
			"grafana_cloud_ips":          grafana.DatasourceCloudIPs(),
			"grafana_cloud_organization": grafana.DatasourceCloudOrganization(),
			"grafana_cloud_stack":        grafana.DatasourceCloudStack(),
		})

		// Datasources that require the OnCall client to exist.
		onCallClientDatasources = addResourcesMetadataValidation(onCallClientPresent, map[string]*schema.Resource{
			"grafana_oncall_user":             grafana.DataSourceOnCallUser(),
			"grafana_oncall_escalation_chain": grafana.DataSourceOnCallEscalationChain(),
			"grafana_oncall_schedule":         grafana.DataSourceOnCallSchedule(),
			"grafana_oncall_slack_channel":    grafana.DataSourceOnCallSlackChannel(),
			"grafana_oncall_action":           grafana.DataSourceOnCallAction(), // deprecated
			"grafana_oncall_outgoing_webhook": grafana.DataSourceOnCallOutgoingWebhook(),
			"grafana_oncall_user_group":       grafana.DataSourceOnCallUserGroup(),
			"grafana_oncall_team":             grafana.DataSourceOnCallTeam(),
		})
	)

	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("GRAFANA_URL", nil),
					RequiredWith: []string{"auth"},
					Description:  "The root URL of a Grafana server. May alternatively be set via the `GRAFANA_URL` environment variable.",
					ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				},
				"auth": {
					Type:         schema.TypeString,
					Optional:     true,
					Sensitive:    true,
					DefaultFunc:  schema.EnvDefaultFunc("GRAFANA_AUTH", nil),
					Description:  "API token or basic auth `username:password`. May alternatively be set via the `GRAFANA_AUTH` environment variable.",
					AtLeastOneOf: []string{"auth", "cloud_api_key", "sm_access_token", "oncall_access_token"},
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
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_RETRIES", 3),
					Description: "The amount of retries to use for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRIES` environment variable.",
				},
				"org_id": {
					Type:        schema.TypeInt,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_ORG_ID", 1),
					Description: "The default organization id to operate on within grafana. For resources that have an `org_id` attribute, the resource-level attribute has priority. May alternatively be set via the `GRAFANA_ORG_ID` environment variable.",
				},
				"tls_key": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_TLS_KEY", nil),
					Description: "Client TLS key file to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_KEY` environment variable.",
				},
				"tls_cert": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_TLS_CERT", nil),
					Description: "Client TLS certificate file to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_CERT` environment variable.",
				},
				"ca_cert": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_CA_CERT", nil),
					Description: "Certificate CA bundle to use to verify the Grafana server's certificate. May alternatively be set via the `GRAFANA_CA_CERT` environment variable.",
				},
				"insecure_skip_verify": {
					Type:        schema.TypeBool,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_INSECURE_SKIP_VERIFY", nil),
					Description: "Skip TLS certificate verification. May alternatively be set via the `GRAFANA_INSECURE_SKIP_VERIFY` environment variable.",
				},

				"cloud_api_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_CLOUD_API_KEY", nil),
					Description: "API key for Grafana Cloud. May alternatively be set via the `GRAFANA_CLOUD_API_KEY` environment variable.",
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
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_SM_ACCESS_TOKEN", nil),
					Description: "A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.",
				},
				"sm_url": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("GRAFANA_SM_URL", "https://synthetic-monitoring-api.grafana.net"),
					Description:  "Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.",
					ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				},
				"store_dashboard_sha256": {
					Type:        schema.TypeBool,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_STORE_DASHBOARD_SHA256", false),
					Description: "Set to true if you want to save only the sha256sum instead of complete dashboard model JSON in the tfstate.",
				},

				"oncall_access_token": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_ONCALL_ACCESS_TOKEN", nil),
					Description: "A Grafana OnCall access token. May alternatively be set via the `GRAFANA_ONCALL_ACCESS_TOKEN` environment variable.",
				},
				"oncall_url": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("GRAFANA_ONCALL_URL", "https://oncall-prod-us-central-0.grafana.net/oncall"),
					Description:  "An Grafana OnCall backend address. May alternatively be set via the `GRAFANA_ONCALL_URL` environment variable.",
					ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				},
			},

			ResourcesMap: mergeResourceMaps(
				map[string]*schema.Resource{
					// Special case, this resource supports both Grafana and Cloud (depending on context)
					"grafana_api_key": grafana.ResourceAPIKey(),
					// This one installs SM on a cloud instance, everything it needs is in its attributes
					"grafana_synthetic_monitoring_installation": grafana.ResourceSyntheticMonitoringInstallation(),
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
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var (
			diags diag.Diagnostics
			err   error
		)
		p.UserAgent("terraform-provider-grafana", version)

		c := &common.Client{}

		if d.Get("auth").(string) != "" && d.Get("url").(string) != "" {
			c.GrafanaAPIURL, c.GrafanaAPIConfig, c.GrafanaAPI, err = createGrafanaClient(d)
			if err != nil {
				return nil, diag.FromErr(err)
			}
			c.MLAPI, err = createMLClient(c.GrafanaAPIURL, c.GrafanaAPIConfig)
			if err != nil {
				return nil, diag.FromErr(err)
			}
		}
		if d.Get("cloud_api_key").(string) != "" {
			c.GrafanaCloudAPI, err = createCloudClient(d)
			if err != nil {
				return nil, diag.FromErr(err)
			}
		}
		c.SMAPIURL = d.Get("sm_url").(string)
		if smToken := d.Get("sm_access_token").(string); smToken != "" {
			c.SMAPI = SMAPI.NewClient(c.SMAPIURL, smToken, nil)
		}
		if d.Get("oncall_access_token").(string) != "" {
			var onCallClient *onCallAPI.Client
			onCallClient, err = createOnCallClient(d)
			if err != nil {
				return nil, diag.FromErr(err)
			}
			onCallClient.UserAgent = p.UserAgent("terraform-provider-grafana", version)
			c.OnCallClient = onCallClient
		}

		grafana.StoreDashboardSHA256 = d.Get("store_dashboard_sha256").(bool)

		return c, diags
	}
}

func createGrafanaClient(d *schema.ResourceData) (string, *gapi.Config, *gapi.Client, error) {
	auth := strings.SplitN(d.Get("auth").(string), ":", 2)
	cli := cleanhttp.DefaultClient()
	transport := cleanhttp.DefaultTransport()
	transport.TLSClientConfig = &tls.Config{}

	// TLS Config
	tlsKey := d.Get("tls_key").(string)
	tlsCert := d.Get("tls_cert").(string)
	caCert := d.Get("ca_cert").(string)
	insecure := d.Get("insecure_skip_verify").(bool)
	if caCert != "" {
		ca, err := os.ReadFile(caCert)
		if err != nil {
			return "", nil, nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		transport.TLSClientConfig.RootCAs = pool
	}
	if tlsKey != "" && tlsCert != "" {
		cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
		if err != nil {
			return "", nil, nil, err
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}
	if insecure {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	apiURL := d.Get("url").(string)
	cli.Transport = logging.NewSubsystemLoggingHTTPTransport("Grafana", transport)
	cfg := gapi.Config{
		Client:     cli,
		NumRetries: d.Get("retries").(int),
	}
	if len(auth) == 2 {
		cfg.BasicAuth = url.UserPassword(auth[0], auth[1])
		cfg.OrgID = int64(d.Get("org_id").(int))
	} else {
		cfg.APIKey = auth[0]
	}

	var err error
	if cfg.HTTPHeaders, err = getHTTPHeadersMap(d); err != nil {
		return "", nil, nil, err
	}

	gclient, err := gapi.New(apiURL, cfg)
	if err != nil {
		return "", nil, nil, err
	}
	return apiURL, &cfg, gclient, nil
}

func createMLClient(url string, grafanaCfg *gapi.Config) (*mlapi.Client, error) {
	mlcfg := mlapi.Config{
		BasicAuth:   grafanaCfg.BasicAuth,
		BearerToken: grafanaCfg.APIKey,
		Client:      grafanaCfg.Client,
		NumRetries:  grafanaCfg.NumRetries,
	}
	mlURL := url
	if !strings.HasSuffix(mlURL, "/") {
		mlURL += "/"
	}
	mlURL += "api/plugins/grafana-ml-app/resources"
	mlclient, err := mlapi.New(mlURL, mlcfg)
	if err != nil {
		return nil, err
	}
	return mlclient, nil
}

func createCloudClient(d *schema.ResourceData) (*gapi.Client, error) {
	cfg := gapi.Config{
		APIKey:     d.Get("cloud_api_key").(string),
		NumRetries: d.Get("retries").(int),
	}

	var err error
	if cfg.HTTPHeaders, err = getHTTPHeadersMap(d); err != nil {
		return nil, err
	}

	return gapi.New(d.Get("cloud_api_url").(string), cfg)
}

func createOnCallClient(d *schema.ResourceData) (*onCallAPI.Client, error) {
	aToken := d.Get("oncall_access_token").(string)
	baseURL := d.Get("oncall_url").(string)
	return onCallAPI.New(baseURL, aToken)
}

func getHTTPHeadersMap(d *schema.ResourceData) (map[string]string, error) {
	headersMap := d.Get("http_headers").(map[string]interface{})
	if len(headersMap) == 0 {
		// We cannot use a DefaultFunc because they do not work on maps
		var err error
		headersMap, err = getJSONMap("GRAFANA_HTTP_HEADERS")
		if err != nil {
			return nil, fmt.Errorf("invalid http_headers config: %w", err)
		}
	}
	if len(headersMap) > 0 {
		headers := make(map[string]string)
		for k, v := range headersMap {
			if v, ok := v.(string); ok {
				headers[k] = v
			}
		}
		return headers, nil
	}
	return map[string]string{}, nil
}

// getJSONMap is a helper function that parses the given environment variable as a JSON object
func getJSONMap(k string) (map[string]interface{}, error) {
	if valStr := os.Getenv(k); valStr != "" {
		var valObj map[string]interface{}
		err := json.Unmarshal([]byte(valStr), &valObj)
		if err != nil {
			return nil, err
		}
		return valObj, nil
	}
	return nil, nil
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
