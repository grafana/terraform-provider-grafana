package grafana

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceTeamPreferences() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/preferences/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/team/)
`,

		CreateContext: UpdateTeamPreferences,
		ReadContext:   ReadTeamPreferences,
		UpdateContext: UpdateTeamPreferences,
		DeleteContext: DeleteTeamPreferences,

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The numeric team ID.",
			},
			"theme": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"light", "dark", ""}, false),
				Description:  "The theme for the specified team. Available themes are `light`, `dark`, or an empty string for the default theme.",
			},
			"home_dashboard_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The numeric ID of the dashboard to display when a team member logs in.",
			},
			"timezone": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"utc", "browser", ""}, false),
				Description:  "The timezone for the specified team. Available values are `utc`, `browser`, or an empty string for the default.",
			},
		},
	}
}

func UpdateTeamPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	teamID := int64(d.Get("team_id").(int))
	theme := d.Get("theme").(string)
	homeDashboardID := int64(d.Get("home_dashboard_id").(int))
	timezone := d.Get("timezone").(string)

	preferences := gapi.Preferences{
		Theme:           theme,
		HomeDashboardID: homeDashboardID,
		Timezone:        timezone,
	}

	err := client.UpdateTeamPreferences(teamID, preferences)
	if err != nil {
		return diag.FromErr(err)
	}

	return ReadTeamPreferences(ctx, d, meta)
}

func ReadTeamPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	teamID := int64(d.Get("team_id").(int))

	preferences, err := client.TeamPreferences(teamID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(teamID, 10))
	d.Set("theme", preferences.Theme)
	d.Set("home_dashboard_id", preferences.HomeDashboardID)
	d.Set("timezone", preferences.Timezone)

	return nil
}

func DeleteTeamPreferences(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// there is no delete call for team preferences. instead we will just remove
	// the specified preferences and go back to the default values. note: if the
	// call fails because the team no longer exists - we'll just ignore the error

	client := meta.(*common.Client).GrafanaAPI

	teamID := int64(d.Get("team_id").(int))
	defaultPreferences := gapi.Preferences{}

	err := client.UpdateTeamPreferences(teamID, defaultPreferences)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
