package grafana

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
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
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_URL", nil),
					Description: "The root URL of a Grafana server. May alternatively be set via the `GRAFANA_URL` environment variable.",
				},
				"auth": {
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_AUTH", nil),
					Description: "API token or basic auth username:password. May alternatively be set via the `GRAFANA_AUTH` environment variable.",
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
				"sm_access_token": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_SM_ACCESS_TOKEN", nil),
					Description: "A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.",
				},
				"sm_url": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("GRAFANA_SM_URL", "https://synthetic-monitoring-api.grafana.net"),
					Description: "Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable.",
				},
			},

			ResourcesMap: map[string]*schema.Resource{

				// Grafana
				"grafana_alert_notification":      ResourceAlertNotification(),
				"grafana_builtin_role_assignment": ResourceBuiltInRoleAssignment(),
				"grafana_dashboard":               ResourceDashboard(),
				"grafana_dashboard_permission":    ResourceDashboardPermission(),
				"grafana_data_source":             ResourceDataSource(),
				"grafana_folder":                  ResourceFolder(),
				"grafana_folder_permission":       ResourceFolderPermission(),
				"grafana_organization":            ResourceOrganization(),
				"grafana_role":                    ResourceRole(),
				"grafana_team":                    ResourceTeam(),
				"grafana_team_preferences":        ResourceTeamPreferences(),
				"grafana_team_external_group":     ResourceTeamExternalGroup(),
				"grafana_user":                    ResourceUser(),

				// Synthetic Monitoring
				"grafana_synthetic_monitoring_check": resourceSyntheticMonitoringCheck(),
				"grafana_synthetic_monitoring_probe": resourceSyntheticMonitoringProbe(),
			},

			DataSourcesMap: map[string]*schema.Resource{

				// Synthetic Monitoring
				"grafana_synthetic_monitoring_probe":  dataSourceSyntheticMonitoringProbe(),
				"grafana_synthetic_monitoring_probes": dataSourceSyntheticMonitoringProbes(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type client struct {
	gapi  *gapi.Client
	smapi *smapi.Client
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {

		var diags diag.Diagnostics
		p.UserAgent("terraform-provider-grafana", version)

		c := &client{}

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
			ca, err := ioutil.ReadFile(caCert)
			if err != nil {
				return nil, diag.FromErr(err)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(ca)
			transport.TLSClientConfig.RootCAs = pool
		}
		if tlsKey != "" && tlsCert != "" {
			cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
			if err != nil {
				return nil, diag.FromErr(err)
			}
			transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
		}
		if insecure {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}

		cli.Transport = logging.NewTransport("Grafana", transport)
		cfg := gapi.Config{
			Client: cli,
			OrgID:  int64(d.Get("org_id").(int)),
		}
		if len(auth) == 2 {
			cfg.BasicAuth = url.UserPassword(auth[0], auth[1])
		} else {
			cfg.APIKey = auth[0]
		}
		gclient, err := gapi.New(d.Get("url").(string), cfg)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		c.gapi = gclient

		smToken := d.Get("sm_access_token").(string)
		smUrl := d.Get("sm_url").(string)
		c.smapi = smapi.NewClient(smUrl, smToken, nil)

		return c, diags
	}
}
