package grafana

import (
	"context"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOrganizationPreferences() *common.Resource {
	schema := &schema.Resource{

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
			"home_dashboard_uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Organization home dashboard UID. This is only available in Grafana 9.0+.",
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

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_organization_preferences",
		common.NewResourceID(common.IntIDField("orgID")),
		schema,
	).WithLister(listerFunction(listOrganizationPreferences))
}

func listOrganizationPreferences(ctx context.Context, client *goapi.GrafanaHTTPAPI, data *ListerData) ([]string, error) {
	orgIDs, err := listOrganizations(ctx, client, data)
	orgIDs = append(orgIDs, "1") // Default org. We can set preferences for it even if it can't be managed otherwise.
	return orgIDs, err
}

func CreateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	_, err := client.OrgPreferences.UpdateOrgPreferences(&models.UpdatePrefsCmd{
		Theme:            d.Get("theme").(string),
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

func ReadOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	id := d.Id() + ":" // Ensure the ID is in the <orgID>:<resourceID> format. A bit hacky but won't survive the migration to plugin framework
	client, _, _ := OAPIClientFromExistingOrgResource(meta, id)

	resp, err := client.OrgPreferences.GetOrgPreferences()
	if err, shouldReturn := common.CheckReadError("organization preferences", d, err); shouldReturn {
		return err
	}
	prefs := resp.Payload

	d.Set("org_id", d.Id())
	d.Set("theme", prefs.Theme)
	d.Set("home_dashboard_uid", prefs.HomeDashboardUID)
	d.Set("timezone", prefs.Timezone)
	d.Set("week_start", prefs.WeekStart)

	return nil
}

func UpdateOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return CreateOrganizationPreferences(ctx, d, meta)
}

func DeleteOrganizationPreferences(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	id := d.Id() + ":" // Ensure the ID is in the <orgID>:<resourceID> format. A bit hacky but won't survive the migration to plugin framework
	client, _, _ := OAPIClientFromExistingOrgResource(meta, id)

	if _, err := client.OrgPreferences.UpdateOrgPreferences(&models.UpdatePrefsCmd{}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
