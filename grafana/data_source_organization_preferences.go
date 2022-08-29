package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceOrganizationPreferences() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/manage-organizations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/preferences/#get-current-org-prefs)
`,
		ReadContext: dataSourceOrganizationPreferencesRead,
		Schema: map[string]*schema.Schema{
			"theme": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Organization theme.",
			},
			"home_dashboard_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The Organization home dashboard ID.",
			},
			"home_dashboard_uid": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The Organization home dashboard UID.",
			},
			"timezone": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Organization timezone.",
			},
			"week_start": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Organization week start.",
			},
			"locale": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Organization locale.",
			},
		},
	}
}

func dataSourceOrganizationPreferencesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	prefs, err := client.OrgPreferences()

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("theme", prefs.Theme)
	d.Set("home_dashboard_id", prefs.HomeDashboardID)
	d.Set("home_dashboard_uid", prefs.HomeDashboardUID)
	d.Set("timezone", prefs.Timezone)
	d.Set("week_start", prefs.WeekStart)
	d.Set("locale", prefs.Locale)

	return nil
}
