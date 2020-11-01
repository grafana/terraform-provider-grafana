package grafana

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_URL", nil),
				Description: "URL of the root of the target Grafana server.",
			},
			"auth": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_AUTH", nil),
				Description: "Credentials for accessing the Grafana API.",
			},
			"org_id": {
				Type:        schema.TypeInt,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_ORG_ID", 1),
				Description: "Organization id for resources",
			},
			"tls_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_TLS_KEY", nil),
				Description: "Client TLS key for accessing the Grafana API.",
			},
			"tls_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_TLS_CERT", nil),
				Description: "Client TLS cert for accessing the Grafana API.",
			},
			"ca_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_CA_CERT", nil),
				Description: "CA cert bundle for validating the Grafana API's certificate.",
			},
			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GRAFANA_INSECURE_SKIP_VERIFY", nil),
				Description: "Skip TLS certificate verification",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"grafana_alert_notification": ResourceAlertNotification(),
			"grafana_dashboard":          ResourceDashboard(),
			"grafana_data_source":        ResourceDataSource(),
			"grafana_folder":             ResourceFolder(),
			"grafana_folder_permission":  ResourceFolderPermission(),
			"grafana_organization":       ResourceOrganization(),
			"grafana_team":               ResourceTeam(),
			"grafana_team_preferences":   ResourceTeamPreferences(),
			"grafana_user":               ResourceUser(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
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
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		transport.TLSClientConfig.RootCAs = pool
	}
	if tlsKey != "" && tlsCert != "" {
		cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
		if err != nil {
			return nil, err
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
	client, err := gapi.New(d.Get("url").(string), cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}
