package grafana

import (
	"context"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceOrganizationPreferences() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/organization-management/)
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
			"org_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization ID. If not set, the Org ID defined in the provider block will be used.",
				ForceNew:    true,
			},
			"theme": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The Organization theme. Available values are `light`, `dark`, or an empty string for the default.",
				ValidateFunc: validation.StringInSlice([]string{"light", "dark", ""}, false),
			},
			"home_dashboard_id": {
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "The Organization home dashboard ID.",
				ConflictsWith: []string{"home_dashboard_uid"},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, uidSet := d.GetOk("home_dashboard_uid")
					return uidSet
				},
			},
			"home_dashboard_uid": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The Organization home dashboard UID. This is only available in Grafana 9.0+.",
				ConflictsWith: []string{"home_dashboard_id"},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, idSet := d.GetOk("home_dashboard_id")
					return idSet
				},
			},
			"timezone": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The Organization timezone. Available values are `utc`, `browser`, or an empty string for the default.",
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
	client, orgID := ClientFromOrgIDAttr(meta, d)

	_, err := client.UpdateAllOrgPreferences(gapi.Preferences{
		Theme:            d.Get("theme").(string),
		HomeDashboardID:  int64(d.Get("home_dashboard_id").(int)),
		HomeDashboardUID: d.Get("home_dashboard_uid").(string),
		Timezone:         d.Get("timezone").(string),
		WeekStart:        d.Get("week_start").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(orgID, 10))

	return ReadOrganizationPreferences(ctx, d, meta)
}

func ReadOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	if id, _ := strconv.ParseInt(d.Id(), 10, 64); id > 0 {
		client = client.WithOrgID(id)
	}

	prefs, err := client.OrgPreferences()

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("theme", prefs.Theme)
	d.Set("home_dashboard_id", prefs.HomeDashboardID)
	d.Set("home_dashboard_uid", prefs.HomeDashboardUID)
	d.Set("timezone", prefs.Timezone)
	d.Set("week_start", prefs.WeekStart)

	return nil
}

func UpdateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return CreateOrganizationPreferences(ctx, d, meta)
}

func DeleteOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	if id, _ := strconv.ParseInt(d.Id(), 10, 64); id > 0 {
		client = client.WithOrgID(id)
	}

	if _, err := client.UpdateAllOrgPreferences(gapi.Preferences{}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
