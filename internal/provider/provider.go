package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/go-openapi/strfmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	gapi "github.com/grafana/grafana-api-golang-client"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/machine-learning-go-client/mlapi"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/resources/machinelearning"
	"github.com/grafana/terraform-provider-grafana/internal/resources/oncall"
	"github.com/grafana/terraform-provider-grafana/internal/resources/slo"
	"github.com/grafana/terraform-provider-grafana/internal/resources/syntheticmonitoring"
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
			"grafana_api_key":                    grafana.ResourceAPIKey(),
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
					Description:  "API token, basic auth in the `username:password` format or `anonymous` (string literal). May alternatively be set via the `GRAFANA_AUTH` environment variable.",
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
				"retry_status_codes": {
					Type:        schema.TypeSet,
					Optional:    true,
					Description: "The status codes to retry on for Grafana API and Grafana Cloud API calls. Use `x` as a digit wildcard. Defaults to 429 and 5xx. May alternatively be set via the `GRAFANA_RETRY_STATUS_CODES` environment variable.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"retry_wait": {
					Type:        schema.TypeInt,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_RETRY_WAIT", 0),
					Description: "The amount of time in seconds to wait between retries for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRY_WAIT` environment variable.",
				},
				"org_id": {
					Type:        schema.TypeInt,
					Optional:    true,
					Deprecated:  "Use the `org_id` attributes on resources instead.",
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_ORG_ID", nil),
					Description: "Deprecated: Use the `org_id` attributes on resources instead.",
				},
				"tls_key": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_TLS_KEY", nil),
					Description: "Client TLS key (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_KEY` environment variable.",
				},
				"tls_cert": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_TLS_CERT", nil),
					Description: "Client TLS certificate (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_CERT` environment variable.",
				},
				"ca_cert": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_CA_CERT", nil),
					Description: "Certificate CA bundle (file path or literal value) to use to verify the Grafana server's certificate. May alternatively be set via the `GRAFANA_CA_CERT` environment variable.",
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
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_SM_ACCESS_TOKEN", nil),
					Description: "A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.",
				},
				"sm_url": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("GRAFANA_SM_URL", "https://synthetic-monitoring-api.grafana.net"),
					Description:  "Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/monitor-public-endpoints/private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.",
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
			c.GrafanaOAPI, err = createGrafanaOAPIClient(c.GrafanaAPIURL, d)
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
		if smToken := d.Get("sm_access_token").(string); smToken != "" {
			retryClient := retryablehttp.NewClient()
			retryClient.RetryMax = d.Get("retries").(int)
			if wait := d.Get("retry_wait").(int); wait > 0 {
				retryClient.RetryWaitMin = time.Second * time.Duration(d.Get("retry_wait").(int))
				retryClient.RetryWaitMax = time.Second * time.Duration(d.Get("retry_wait").(int))
			}

			c.SMAPI = SMAPI.NewClient(d.Get("sm_url").(string), smToken, retryClient.StandardClient())
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
	cli := cleanhttp.DefaultClient()
	transport := cleanhttp.DefaultTransport()
	// limiting the amount of concurrent HTTP connections from the provider
	// makes it not overload the API and DB
	transport.MaxConnsPerHost = 2

	tlsClientConfig, err := parseTLSconfig(d)
	if err != nil {
		return "", nil, nil, err
	}
	transport.TLSClientConfig = tlsClientConfig

	apiURL := d.Get("url").(string)
	cli.Transport = logging.NewSubsystemLoggingHTTPTransport("Grafana", transport)

	userInfo, orgID, apiKey, err := parseAuth(d)
	if err != nil {
		return "", nil, nil, err
	}

	cfg := gapi.Config{
		Client:       cli,
		NumRetries:   d.Get("retries").(int),
		RetryTimeout: time.Second * time.Duration(d.Get("retry_wait").(int)),
		BasicAuth:    userInfo,
		OrgID:        orgID,
		APIKey:       apiKey,
	}

	if v, ok := d.GetOk("retry_status_codes"); ok {
		cfg.RetryStatusCodes = common.SetToStringSlice(v.(*schema.Set))
	}

	if cfg.HTTPHeaders, err = getHTTPHeadersMap(d); err != nil {
		return "", nil, nil, err
	}

	gclient, err := gapi.New(apiURL, cfg)
	if err != nil {
		return "", nil, nil, err
	}
	return apiURL, &cfg, gclient, nil
}

func createGrafanaOAPIClient(apiURL string, d *schema.ResourceData) (*goapi.GrafanaHTTPAPI, error) {
	tlsClientConfig, err := parseTLSconfig(d)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API url: %v", err.Error())
	}

	userInfo, orgID, APIKey, err := parseAuth(d)
	if err != nil {
		return nil, err
	}

	cfg := goapi.TransportConfig{
		Host:       u.Host,
		BasePath:   "/api",
		Schemes:    []string{u.Scheme},
		NumRetries: d.Get("retries").(int),
		TLSConfig:  tlsClientConfig,
		BasicAuth:  userInfo,
		OrgID:      orgID,
		APIKey:     APIKey,
	}

	return goapi.NewHTTPClientWithConfig(strfmt.Default, &cfg), nil
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
		APIKey:       d.Get("cloud_api_key").(string),
		NumRetries:   d.Get("retries").(int),
		RetryTimeout: time.Second * time.Duration(d.Get("retry_wait").(int)),
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

// Sets a custom HTTP Header on all requests coming from the Grafana Terraform Provider to Grafana-Terraform-Provider: true
// in addition to any headers set within the `http_headers` field or the `GRAFANA_HTTP_HEADERS` environment variable
func getHTTPHeadersMap(d *schema.ResourceData) (map[string]string, error) {
	headers := map[string]string{"Grafana-Terraform-Provider": "true"}
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
		for k, v := range headersMap {
			if v, ok := v.(string); ok {
				headers[k] = v
			}
		}
	}

	return headers, nil
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

func createTempFileIfLiteral(value string) (path string, tempFile bool, err error) {
	if value == "" {
		return "", false, nil
	}

	if _, err := os.Stat(value); errors.Is(err, os.ErrNotExist) {
		// value is not a file path, assume it's a literal
		f, err := os.CreateTemp("", "grafana-provider-tls")
		if err != nil {
			return "", false, err
		}
		if _, err := f.WriteString(value); err != nil {
			return "", false, err
		}
		if err := f.Close(); err != nil {
			return "", false, err
		}
		return f.Name(), true, nil
	}

	return value, false, nil
}

func parseAuth(d *schema.ResourceData) (*url.Userinfo, int64, string, error) {
	auth := strings.SplitN(d.Get("auth").(string), ":", 2)
	orgID := 1
	if v, ok := d.GetOk("org_id"); ok {
		orgID = v.(int)
	}

	if len(auth) == 2 {
		return url.UserPassword(auth[0], auth[1]), int64(orgID), "", nil
	} else if auth[0] != "anonymous" {
		if orgID > 1 {
			return nil, 0, "", fmt.Errorf("org_id is only supported with basic auth. API keys are already org-scoped")
		}
		return nil, 0, auth[0], nil
	}
	return nil, 0, "", nil
}

func parseTLSconfig(d *schema.ResourceData) (*tls.Config, error) {
	tlsClientConfig := &tls.Config{}

	tlsKeyFile, tempFile, err := createTempFileIfLiteral(d.Get("tls_key").(string))
	if err != nil {
		return nil, err
	}
	if tempFile {
		defer os.Remove(tlsKeyFile)
	}
	tlsCertFile, tempFile, err := createTempFileIfLiteral(d.Get("tls_cert").(string))
	if err != nil {
		return nil, err
	}
	if tempFile {
		defer os.Remove(tlsCertFile)
	}
	caCertFile, tempFile, err := createTempFileIfLiteral(d.Get("ca_cert").(string))
	if err != nil {
		return nil, err
	}
	if tempFile {
		defer os.Remove(caCertFile)
	}

	insecure := d.Get("insecure_skip_verify").(bool)
	if caCertFile != "" {
		ca, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		tlsClientConfig.RootCAs = pool
	}
	if tlsKeyFile != "" && tlsCertFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
		if err != nil {
			return nil, err
		}
		tlsClientConfig.Certificates = []tls.Certificate{cert}
	}
	if insecure {
		tlsClientConfig.InsecureSkipVerify = true
	}

	return tlsClientConfig, nil
}
