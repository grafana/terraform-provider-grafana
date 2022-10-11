package grafana

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceOrganizationPreferences() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/manage-organizations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/preferences/#get-current-org-prefs)
`,

		CreateContext: CreateOrganizationPreferences,
		ReadContext:   ReadOrganizationPreferences,
		UpdateContext: UpdateOrganizationPreferences,
		DeleteContext: DeleteOrganizationPreferences,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"theme": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The Organization theme.",
				ValidateFunc: validation.StringInSlice([]string{"light", "dark", ""}, false),
			},
			"home_dashboard_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The Organization home dashboard ID.",
			},
			"home_dashboard_uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization home dashboard UID.",
			},
			"timezone": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The Organization timezone.",
				ValidateFunc: validation.StringInSlice([]string{"utc", "browser", ""}, false),
			},
			// TODO: add validation?
			"week_start": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization week start.",
			},
		},
	}
}

func CreateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	_, err := client.UpdateAllOrgPreferences(gapi.Preferences{
		Theme:            d.Get("theme").(string),
		HomeDashboardID:  d.Get("home_dashboard_id").(int64),
		HomeDashboardUID: d.Get("home_dashboard_uid").(string),
		Timezone:         d.Get("timezone").(string),
		WeekStart:        d.Get("week_start").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("organization_preferences")

	return ReadOrganizationPreferences(ctx, d, meta)
}

func ReadOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	d.SetId("organization_preferences")

	return nil
}

func UpdateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return CreateOrganizationPreferences(ctx, d, meta)
}

func DeleteOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	if _, err := client.UpdateAllOrgPreferences(gapi.Preferences{}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
