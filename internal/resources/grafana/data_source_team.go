package grafana

import (
	"context"
	"fmt"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceTeam() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/team/)
`,
		ReadContext: dataSourceTeamRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Grafana team.",
			},
		},
	}
}

func findTeamWithName(client *gapi.Client, name string) (*gapi.Team, error) {
	searchTeam, err := client.SearchTeam(name)
	if err != nil {
		return nil, err
	}

	for _, f := range searchTeam.Teams {
		if f.Name == name {
			// Query the team by ID, that API has additional information
			return client.Team(f.ID)
		}
	}

	return nil, fmt.Errorf("no team with name %q", name)
}

func dataSourceTeamRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	name := d.Get("name").(string)
	team, err := findTeamWithName(client, name)

	if err != nil {
		return diag.FromErr(err)
	}

	id := strconv.FormatInt(team.ID, 10)
	d.SetId(id)
	d.Set("name", team.Name)

	return nil
}
