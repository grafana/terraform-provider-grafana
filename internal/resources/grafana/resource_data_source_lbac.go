package grafana

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLBACRule() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/data_source/)

The required arguments for this resource vary depending on the type of data
source selected (via the 'type' argument).

Example usage:
resource "grafana_lbac_rule" "example" {
  datasource_uid = "some-unique-datasource-uid"
  team_id        = ["team1", "team2"]
  rules          = [
    "{ foo != \"bar\", foo !~ \"baz\" }",
    "{ foo = \"qux\", bar ~ \"quux\" }"
  ]
}
`,
		CreateContext: resourceTeamLBACRuleCreate,
		ReadContext:   resourceTeamLBACRuleRead,
		UpdateContext: resourceTeamLBACRuleUpdate,

		Schema: map[string]*schema.Schema{
			"datasource_uid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique identifier of the Grafana datasource.",
			},
			"team_id": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of team IDs for which LBAC rules are set.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"rules": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of LBAC rules for the teams.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
		"grafana_data_source_lbac_team",
		nil,
		schema,
	)
}

func resourceTeamLBACRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*common.Client).GrafanaAPI.Clone()
	datasourceUID := d.Get("datasource_uid").(string)
	teamID := d.Get("team_id").(string)
	rules := d.Get("rules").([]interface{})

	lbacRules := []models.TeamL{
		{
			TeamID: teamID,
			Rules:  convertInterfaceSliceToStringSlice(rules),
		},
	}

	command := &models.UpdateDataSourceTeamLBACRulesCommand{
		TeamLBACRules: lbacRules,
	}

	_, err := c.Enterprise.UpdateTeamLBACRulesAPI(datasourceUID, command)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(datasourceUID + ":" + teamID)
	return resourceTeamLBACRuleRead(ctx, d, m)
}

func resourceTeamLBACRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*common.Client).GrafanaAPI.Clone()
	datasourceUID, teamID := parseCompositeID(d.Id())

	resp, err := c.Enterprise.GetTeamLBACRulesAPI(datasourceUID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = resp.Payload.UnmarshalBinary()
	if err != nil {
		return diag.FromErr(err)
	}
	fmt.Printf("%+v", resp.GetPayload().Message)
	d.SetId("") // Resource not found, mark it for recreation
	return nil
}

func resourceTeamLBACRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceTeamLBACRuleCreate(ctx, d, m)
}

func convertInterfaceSliceToStringSlice(input []interface{}) []string {
	output := make([]string, len(input))
	for i, v := range input {
		output[i] = v.(string)
	}
	return output
}

func parseCompositeID(compositeID string) (string, string) {
	parts := strings.SplitN(compositeID, ":", 2)
	return parts[0], parts[1]
}
