package grafana

import (
	"strings"
	"net/url"

	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/go-cleanhttp"

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
		},

		ResourcesMap: map[string]*schema.Resource{
			"grafana_alert_notification": ResourceAlertNotification(),
			"grafana_dashboard":          ResourceDashboard(),
			"grafana_data_source":        ResourceDataSource(),
			"grafana_folder":             ResourceFolder(),
			"grafana_organization":       ResourceOrganization(),
			"grafana_team":               ResourceTeam(),
			"grafana_user":               ResourceUser(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	splitAuth := strings.Split(d.Get("auth").(string), ":")
	cli := cleanhttp.DefaultClient()
	cli.Transport = logging.NewTransport("Grafana", cli.Transport)
	client, err := gapi.New(
		d.Get("url").(string),
		gapi.Config{
			BasicAuth: url.UserPassword(splitAuth[0], splitAuth[1]),
			Client: cli,
		},
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
