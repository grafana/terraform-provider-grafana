package grafana

import (
	"context"

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
		Schema: common.CloneResourceSchemaForDatasource(ResourceTeam(), map[string]*schema.Schema{
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
}

func dataSourceTeamRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := ClientFromNewOrgResource(meta, d)
	name := d.Get("name").(string)
	searchTeam, err := client.SearchTeam(name)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, r := range searchTeam.Teams {
		if r.Name == name {
			return readTeamFromID(client, r.ID, d, d.Get("read_team_sync").(bool))
		}
	}

	return diag.Errorf("no team with name %q", name)
}
