package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/models"
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
			"org_id": orgIDAttribute(),
			"theme": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The Organization theme. Available values are `light`, `dark`, `system`, or an empty string for the default.",
				ValidateFunc: validation.StringInSlice([]string{"light", "dark", "system", ""}, false),
			},
			"home_dashboard_id": {
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "The Organization home dashboard ID. Deprecated: Use `home_dashboard_uid` instead.",
				ConflictsWith: []string{"home_dashboard_uid"},
				Deprecated:    "Use `home_dashboard_uid` instead.",
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
			"week_start": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The Organization week start day. Available values are `sunday`, `monday`, `saturday`, or an empty string for the default.",
				ValidateFunc: validation.StringInSlice([]string{"sunday", "monday", "saturday", ""}, false),
				Default:      "",
			},
		},
	}
}

func CreateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	_, err := client.OrgPreferences.UpdateOrgPreferences(&models.UpdatePrefsCmd{
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
	client := OAPIGlobalClient(meta)
	if id, _ := strconv.ParseInt(d.Id(), 10, 64); id > 0 {
		client = client.WithOrgID(id)
	}

	resp, err := client.OrgPreferences.GetOrgPreferences()
	if err, shouldReturn := common.CheckReadError("organization preferences", d, err); shouldReturn {
		return err
	}
	prefs := resp.Payload

	d.Set("org_id", d.Id())
	d.Set("theme", prefs.Theme)
	d.Set("home_dashboard_id", int(prefs.HomeDashboardID))
	d.Set("home_dashboard_uid", prefs.HomeDashboardUID)
	d.Set("timezone", prefs.Timezone)
	d.Set("week_start", prefs.WeekStart)

	return nil
}

func UpdateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return CreateOrganizationPreferences(ctx, d, meta)
}

func DeleteOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta)
	if id, _ := strconv.ParseInt(d.Id(), 10, 64); id > 0 {
		client = client.WithOrgID(id)
	}

	if _, err := client.OrgPreferences.UpdateOrgPreferences(&models.UpdatePrefsCmd{}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
