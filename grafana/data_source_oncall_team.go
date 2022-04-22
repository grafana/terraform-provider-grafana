package grafana

import (
	"errors"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceOnCallTeam() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOnCallTeamRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The team name.",
			},
			"email": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"avatar_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceOnCallTeamRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("Grafana OnCall api client is not configured")
	}
	options := &onCallAPI.ListTeamOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	teamsResponse, _, err := client.Teams.ListTeams(options)
	if err != nil {
		return err
	}

	if len(teamsResponse.Teams) == 0 {
		return fmt.Errorf("couldn't find a team matching: %s", options.Name)
	} else if len(teamsResponse.Teams) != 1 {
		return fmt.Errorf("more than one team found matching: %s", options.Name)
	}

	team := teamsResponse.Teams[0]

	d.Set("name", team.Name)
	d.Set("email", team.Email)
	d.Set("avatar_url", team.AvatarUrl)

	d.SetId(team.ID)

	return nil
}
