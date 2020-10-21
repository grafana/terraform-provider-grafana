package grafana

import (
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
	client, err := gapi.New(
		d.Get("auth").(string),
		d.Get("url").(string),
	)
	if err != nil {
		return nil, err
	}

	client.Transport = logging.NewTransport("Grafana", client.Transport)

	return client, nil
}
