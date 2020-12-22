package grafana

import (
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
		},

		ResourcesMap: map[string]*schema.Resource{
			"grafana_alert_notification":   ResourceAlertNotification(),
			"grafana_dashboard":            ResourceDashboard(),
			"grafana_dashboard_permission": ResourceDashboardPermission(),
			"grafana_data_source":          ResourceDataSource(),
			"grafana_folder":               ResourceFolder(),
			"grafana_folder_permission":    ResourceFolderPermission(),
			"grafana_organization":         ResourceOrganization(),
			"grafana_team":                 ResourceTeam(),
			"grafana_team_preferences":     ResourceTeamPreferences(),
			"grafana_user":                 ResourceUser(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	auth := strings.SplitN(d.Get("auth").(string), ":", 2)
	cli := cleanhttp.DefaultClient()
	cli.Transport = logging.NewTransport("Grafana", cli.Transport)
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
