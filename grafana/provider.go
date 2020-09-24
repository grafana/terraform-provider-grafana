package grafana

import (
	"encoding/json"
	"os"

	"github.com/hashicorp/terraform/helper/logging"
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
			"headers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Optional. HTTP headers mapping keys to values used for accessing the Grafana API.",
				DefaultFunc: EnvDefaultJsonFunc("GRAFANA_HTTP_HEADERS", nil),
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
	headersObj := d.Get("headers").(map[string]interface{})
	if headersObj != nil && len(headersObj) == 0 {
		// Workaround for a bug when DefaultFunc returns a TypeMap
		headersObjAbs, _ := EnvDefaultJsonFunc("GRAFANA_HTTP_HEADERS", nil)()
		headersObj = headersObjAbs.(map[string]interface{})
	}

	// Convert headers from map[string]interface{} to map[string]string
	headers := make(map[string]string)
	if headersObj != nil {
		for k, v := range headersObj {
			switch v := v.(type) {
			case string:
				headers[k] = v
			}
		}
	}

	client, err := gapi.New(
		d.Get("auth").(string),
		d.Get("url").(string),
		headers,
	)
	if err != nil {
		return nil, err
	}

	client.Transport = logging.NewTransport("Grafana", client.Transport)

	return client, nil
}

// EnvDefaultJsonFunc is a helper function that parses the given environment
// variable as a JSON object, or returns the default value otherwise.
func EnvDefaultJsonFunc(k string, dv interface{}) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if valStr := os.Getenv(k); valStr != "" {
			var valObj map[string]interface{}
			err := json.Unmarshal([]byte(valStr), &valObj)
			if err != nil {
				return nil, err
			}
			return valObj, nil
		}

		return dv, nil
	}
}
