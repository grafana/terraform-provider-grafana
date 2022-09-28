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
				Type:        schema.TypeString,
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

	// In current implementation generateOrgPrefsIDSha() returns a sha sum of all the resource & data
	// source attributes, and is also used by resource_organization_preferences.
	//
	// TODO: is it problematic that every organization_preference data source will have the same ID?
	//
	// According to @julienduchesne (https://github.com/grafana/terraform-provider-grafana/pull/583/files/b261189cf70ae4c076d9319d83abda2a959e5112#r944357467) ...
	// "The ID should be declarative because it needs to be unique.
	// For a datasource like this, it's usually the combination of all entry parameters because you
	// will typically not have the same datasource twice with the same parameters"
	//
	// However, in this instance, the data source does not accept any parameters; they are all computed.
	// So, what would be a reasonable way to calculate its ID?
	d.SetId(generateOrgPrefsIDSha())

	return nil
}
