package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceOrganizationPreferences() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/organization-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/preferences/#get-current-org-prefs)
`,
		ReadContext: dataSourceOrganizationPreferencesRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceOrganizationPreferences().Schema, map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_organization_preferences", schema)
}

func dataSourceOrganizationPreferencesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	resp, err := client.OrgPreferences.GetOrgPreferences()
	if err != nil {
		return diag.FromErr(err)
	}

	prefs := resp.Payload
	d.Set("theme", prefs.Theme)
	d.Set("home_dashboard_uid", prefs.HomeDashboardUID)
	d.Set("timezone", prefs.Timezone)
	d.Set("week_start", prefs.WeekStart)

	d.SetId("organization_preferences" + strconv.FormatInt(orgID, 10))

	return nil
}
