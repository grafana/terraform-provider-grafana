package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceTeam() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOnCallTeamRead,
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

func dataSourceOnCallTeamRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient
	options := &onCallAPI.ListTeamOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	teamsResponse, _, err := client.Teams.ListTeams(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(teamsResponse.Teams) == 0 {
		return diag.Errorf("couldn't find a team matching: %s", options.Name)
	} else if len(teamsResponse.Teams) != 1 {
		return diag.Errorf("more than one team found matching: %s", options.Name)
	}

	team := teamsResponse.Teams[0]

	d.Set("name", team.Name)
	d.Set("email", team.Email)
	d.Set("avatar_url", team.AvatarUrl)

	d.SetId(team.ID)

	return nil
}
