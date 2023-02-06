package grafana

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/internal/common"
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
		Schema:      common.CloneResourceSchemaForDatasource(ResourceOrganizationPreferences(), map[string]*schema.Schema{}),
	}
}

func dataSourceOrganizationPreferencesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	prefs, err := client.OrgPreferences()

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("theme", prefs.Theme)
	d.Set("home_dashboard_id", prefs.HomeDashboardID)
	d.Set("home_dashboard_uid", prefs.HomeDashboardUID)
	d.Set("timezone", prefs.Timezone)
	d.Set("week_start", prefs.WeekStart)

	d.SetId("organization_preferences")

	return nil
}
