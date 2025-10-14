package oncall

import (
	"context"
	"fmt"
	"time"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTeam() *common.DataSource {
	schema := &schema.Resource{
		ReadContext: withClient[schema.ReadContextFunc](dataSourceTeamRead),
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
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_team", schema)
}

func dataSourceTeamRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	var team *onCallAPI.Team

	// Retry because the team might not be immediately available
	err := retry.RetryContext(ctx, 1*time.Minute, func() *retry.RetryError {
		options := &onCallAPI.ListTeamOptions{}
		nameData := d.Get("name").(string)

		options.Name = nameData

		teamsResponse, _, err := client.Teams.ListTeams(options)
		if err != nil {
			return retry.NonRetryableError(err)
		}

		if len(teamsResponse.Teams) == 0 {
			return retry.RetryableError(fmt.Errorf("couldn't find a team matching: %s", options.Name))
		} else if len(teamsResponse.Teams) != 1 {
			return retry.NonRetryableError(fmt.Errorf("more than one team found matching: %s", options.Name))
		}

		team = teamsResponse.Teams[0]
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", team.Name)
	d.Set("email", team.Email)
	d.Set("avatar_url", team.AvatarUrl)

	d.SetId(team.ID)

	return nil
}
