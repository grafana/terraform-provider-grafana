package grafana

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/machine-learning-go-client/mlapi"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
)

var (
	idRegexp             = regexp.MustCompile(`^\d+$`)
	uidRegexp            = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	emailRegexp          = regexp.MustCompile(`.+\@.+\..+`)
	sha256Regexp         = regexp.MustCompile(`^[A-Fa-f0-9]{64}$`)
	storeDashboardSHA256 bool
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
					Description:  "API token or basic auth username:password. May alternatively be set via the `GRAFANA_AUTH` environment variable.",
					AtLeastOneOf: []string{"auth", "cloud_api_key", "sm_access_token", "oncall_access_token"},
				},
				"http_headers": {
					Type:        schema.TypeMap,
					Optional:    true,
					Sensitive:   true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "Optional. HTTP headers mapping keys to values used for accessing the Grafana API. May alternatively be set via the `GRAFANA_HTTP_HEADERS` environment variable in JSON format.",
				},
				"retries": {
					Type:        schema.TypeInt,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_RETRIES", 3),
					Description: "The amount of retries to use for Grafana API calls. May alternatively be set via the `GRAFANA_RETRIES` environment variable.",
				},
				"org_id": {
					Type:        schema.TypeInt,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_ORG_ID", 1),
					Description: "The organization id to operate on within grafana. May alternatively be set via the `GRAFANA_ORG_ID` environment variable.",
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
					Description:  "Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable.",
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
					DefaultFunc:  schema.EnvDefaultFunc("GRAFANA_ONCALL_URL", "https://a-prod-us-central-0.grafana.net/"),
					Description:  "An Grafana OnCall backend address. May alternatively be set via the `GRAFANA_ONCALL_URL` environment variable.",
					ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				},
			},

			ResourcesMap: map[string]*schema.Resource{
				// Grafana
				"grafana_api_key":                ResourceAPIKey(),
				"grafana_alert_notification":     ResourceAlertNotification(),
				"grafana_dashboard":              ResourceDashboard(),
				"grafana_dashboard_permission":   ResourceDashboardPermission(),
				"grafana_data_source":            ResourceDataSource(),
				"grafana_data_source_permission": ResourceDatasourcePermission(),
				"grafana_folder":                 ResourceFolder(),
				"grafana_folder_permission":      ResourceFolderPermission(),
				"grafana_library_panel":          ResourceLibraryPanel(),
				"grafana_organization":           ResourceOrganization(),
				"grafana_playlist":               ResourcePlaylist(),
				"grafana_report":                 ResourceReport(),
				"grafana_role":                   ResourceRole(),
				"grafana_team":                   ResourceTeam(),
				"grafana_team_preferences":       ResourceTeamPreferences(),
				"grafana_team_external_group":    ResourceTeamExternalGroup(),
				"grafana_user":                   ResourceUser(),

				// Cloud
				"grafana_cloud_api_key":             ResourceCloudAPIKey(),
				"grafana_cloud_plugin_installation": ResourceCloudPluginInstallation(),
				"grafana_cloud_stack":               ResourceCloudStack(),

				// Synthetic Monitoring
				"grafana_synthetic_monitoring_check":        ResourceSyntheticMonitoringCheck(),
				"grafana_synthetic_monitoring_probe":        ResourceSyntheticMonitoringProbe(),
				"grafana_synthetic_monitoring_installation": ResourceSyntheticMonitoringInstallation(),

				// Machine Learning
				"grafana_machine_learning_job": ResourceMachineLearningJob(),

				// OnCall
				"grafana_oncall_integration":      ResourceOnCallIntegration(),
				"grafana_oncall_route":            ResourceOnCallRoute(),
				"grafana_oncall_escalation_chain": ResourceOnCallEscalationChain(),
				"grafana_oncall_escalation":       ResourceOnCallEscalation(),
				"grafana_oncall_on_call_shift":    ResourceOnCallOnCallShift(),
				"grafana_oncall_schedule":         ResourceOnCallSchedule(),
			},

			DataSourcesMap: map[string]*schema.Resource{
				// Grafana
				"grafana_dashboard":     DatasourceDashboard(),
				"grafana_dashboards":    DatasourceDashboards(),
				"grafana_folder":        DatasourceFolder(),
				"grafana_library_panel": DatasourceLibraryPanel(),
				"grafana_user":          DatasourceUser(),

				// Cloud
				"grafana_cloud_stack": DatasourceCloudStack(),

				// Synthetic Monitoring
				"grafana_synthetic_monitoring_probe":  DatasourceSyntheticMonitoringProbe(),
				"grafana_synthetic_monitoring_probes": DatasourceSyntheticMonitoringProbes(),

				// OnCall
				"grafana_oncall_user":             DataSourceOnCallUser(),
				"grafana_oncall_escalation_chain": DataSourceOnCallEscalationChain(),
				"grafana_oncall_schedule":         DataSourceOnCallSchedule(),
				"grafana_oncall_slack_channel":    DataSourceOnCallSlackChannel(),
				"grafana_oncall_action":           DataSourceOnCallAction(),
				"grafana_oncall_user_group":       DataSourceOnCallUserGroup(),
				"grafana_oncall_team":             DataSourceOnCallTeam(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type client struct {
	gapiURL    string
	gapi       *gapi.Client
	gapiConfig *gapi.Config
	gcloudapi  *gapi.Client

	smapi *smapi.Client
	smURL string

	mlapi *mlapi.Client

	onCallAPI *onCallAPI.Client
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var (
			diags diag.Diagnostics
			err   error
		)
		p.UserAgent("terraform-provider-grafana", version)

		c := &client{}

		c.gapiURL, c.gapiConfig, c.gapi, err = createGrafanaClient(d)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		c.gcloudapi, err = createCloudClient(d)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		c.mlapi, err = createMLClient(c.gapiURL, c.gapiConfig)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		c.smURL, c.smapi = createSMClient(d)
		if d.Get("oncall_access_token").(string) != "" {
			c.onCallAPI, err = createOnCallClient(d)
			if err != nil {
				return nil, diag.FromErr(err)
			}
		}

		storeDashboardSHA256 = d.Get("store_dashboard_sha256").(bool)

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
	cli.Transport = logging.NewTransport("Grafana", transport)
	cfg := gapi.Config{
		Client:     cli,
		NumRetries: d.Get("retries").(int),
		OrgID:      int64(d.Get("org_id").(int)),
	}
	if len(auth) == 2 {
		cfg.BasicAuth = url.UserPassword(auth[0], auth[1])
	} else {
		cfg.APIKey = auth[0]
	}

	headersMap := d.Get("http_headers").(map[string]interface{})
	if headersMap != nil && len(headersMap) == 0 {
		// We cannot use a DefaultFunc because they do not work on maps
		var err error
		headersMap, err = getJSONMap("GRAFANA_HTTP_HEADERS")
		if err != nil {
			return "", nil, nil, fmt.Errorf("invalid http_headers config: %w", err)
		}
	}
	if len(headersMap) > 0 {
		headers := make(map[string]string)
		for k, v := range headersMap {
			if v, ok := v.(string); ok {
				headers[k] = v
			}
		}
		cfg.HTTPHeaders = headers
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
		APIKey: d.Get("cloud_api_key").(string),
	}
	return gapi.New(d.Get("cloud_api_url").(string), cfg)
}

func createSMClient(d *schema.ResourceData) (string, *smapi.Client) {
	smToken := d.Get("sm_access_token").(string)
	smURL := d.Get("sm_url").(string)
	return smURL, smapi.NewClient(smURL, smToken, nil)
}

func createOnCallClient(d *schema.ResourceData) (*onCallAPI.Client, error) {
	aToken := d.Get("oncall_access_token").(string)
	base_url := d.Get("oncall_url").(string)
	return onCallAPI.New(base_url, aToken)
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
