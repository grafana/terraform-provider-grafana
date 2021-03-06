package grafana

import (
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceTeamPreferences() *schema.Resource {
	return &schema.Resource{
		Create: UpdateTeamPreferences,
		Read:   ReadTeamPreferences,
		Update: UpdateTeamPreferences,
		Delete: DeleteTeamPreferences,

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"theme": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"light", "dark", ""}, false),
			},
			"home_dashboard_id": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"timezone": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"utc", "browser", ""}, false),
			},
		},
	}
}

func UpdateTeamPreferences(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

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
		return err
	}

	return ReadTeamPreferences(d, meta)
}

func ReadTeamPreferences(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	teamID := int64(d.Get("team_id").(int))

	preferences, err := client.TeamPreferences(teamID)
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(teamID, 10))
	d.Set("theme", preferences.Theme)
	d.Set("home_dashboard_id", preferences.HomeDashboardID)
	d.Set("timezone", preferences.Timezone)

	return nil
}

func DeleteTeamPreferences(d *schema.ResourceData, meta interface{}) error {
	//there is no delete call for team preferences. instead we will just remove
	//the specified preferences and go back to the default values. note: if the
	//call fails because the team no longer exists - we'll just ignore the error

	client := meta.(*gapi.Client)

	teamID := int64(d.Get("team_id").(int))
	defaultPreferences := gapi.Preferences{}

	err := client.UpdateTeamPreferences(teamID, defaultPreferences)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			d.SetId("")
			return nil
		}
		return err
	}

	return nil
}
