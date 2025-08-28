package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceTeam() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/team/)
`,
		ReadContext: dataSourceTeamRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceTeam().Schema, map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Grafana team",
			},
			"read_team_sync": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to read the team sync settings. This is only available in Grafana Enterprise.",
			},
			"ignore_externally_synced_members": nil,
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_team", schema)
}

func dataSourceTeamRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	name := d.Get("name").(string)

	params := teams.NewSearchTeamsParams().WithName(&name)
	resp, err := client.Teams.SearchTeams(params)
	if err != nil {
		return diag.FromErr(err)
	}
	searchTeam := resp.GetPayload()

	for _, r := range searchTeam.Teams {
		if r.Name == name {
			return readTeamFromID(client, r.ID, d, d.Get("read_team_sync").(bool))
		}
	}

	return diag.Errorf("no team with name %q", name)
}
