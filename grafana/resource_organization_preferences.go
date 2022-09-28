package grafana

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization theme.",
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
			// TODO: add validation?
			"timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization timezone.",
			},
			// TODO: add validation?
			"week_start": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization week start.",
			},
			// TODO: add validation?
			"locale": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization locale.",
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
		Locale:           d.Get("locale").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(generateOrgPrefsIDSha())

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
	d.Set("locale", prefs.Locale)

	d.SetId(generateOrgPrefsIDSha())

	return nil
}

func UpdateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	orgID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if d.HasChange("name") {
		name := d.Get("name").(string)
		err := client.UpdateOrg(orgID, name)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if err := UpdateUsers(d, meta); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func DeleteOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	orgID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if err := client.DeleteOrg(orgID); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

// TODO: is it problematic that every org preferences will have the same ID?
// Because the `GET /api/org/preferences` endpoint operates on the current org
// and returns no org ID, it's unclear how best to uniquely ID the
// resource_organization_preferences resource.
// Could/should we look up the current org and use its ID?
func generateOrgPrefsIDSha() string {
	sha := sha256.Sum256([]byte(strings.Join([]string{
		"theme",
		"home_dashboard_id",
		"home_dashboard_uid",
		"timezone",
		"week_start",
		"locale",
	}, "-")))

	return fmt.Sprintf("%x", sha)
}
