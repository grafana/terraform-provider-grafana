package grafana

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	gapi "github.com/nytm/go-grafana-api"
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
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	return gapi.New(
		d.Get("auth").(string),
		d.Get("url").(string),
	)
}
