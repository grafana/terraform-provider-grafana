package oncall

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceIntegration() *common.DataSource {
	schema := &schema.Resource{
		Description: `
Looks up an integration by ID, or by any combination of ` + "`name`" + `, ` + "`team_id`" + ` and ` + "`type`" + `. Exactly one integration must match.

To look up the direct paging integration of a team, set ` + "`team_id`" + ` and ` + "`type = \"direct_paging\"`" + `. A team has at most one.

* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/integrations/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceIntegrationRead),
		Schema: map[string]*schema.Schema{
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				Description:   "The integration ID. Conflicts with `name`, `team_id` and `type`.",
				AtLeastOneOf:  []string{"id", "name", "team_id", "type"},
				ConflictsWith: []string{"name", "team_id", "type"},
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The integration name. When set, only the integration with this exact name matches.",
			},
			"team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The ID of the OnCall team the integration belongs to (using the `grafana_oncall_team` datasource). When set, only integrations of this team match. Set it to `\"null\"` to match only integrations that do not belong to a team.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The type of integration, for example `direct_paging`. When set, only integrations of this type match.",
			},
			"link": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The link for the integration.",
			},
			"inbound_email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The inbound email for the integration. Only available for integration type `inbound_email`.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_integration", schema)
}

func dataSourceIntegrationRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	var integration *onCallAPI.Integration
	if integrationID := d.Get("id").(string); integrationID != "" {
		integrationResponse, r, err := client.Integrations.GetIntegration(integrationID, &onCallAPI.GetIntegrationOptions{})
		if err != nil {
			if r != nil && r.StatusCode == http.StatusNotFound {
				return diag.Errorf("no integration exists with ID %q", integrationID)
			}
			return diag.FromErr(err)
		}
		integration = integrationResponse
	} else {
		var diags diag.Diagnostics
		integration, diags = findIntegration(client, d.Get("name").(string), d.Get("team_id").(string), d.Get("type").(string))
		if diags != nil {
			return diags
		}
	}

	d.SetId(integration.ID)
	d.Set("id", integration.ID)
	d.Set("name", integration.Name)
	d.Set("team_id", integration.TeamId)
	d.Set("type", integration.Type)
	d.Set("link", integration.Link)
	d.Set("inbound_email", integration.InboundEmail)
	return nil
}

// findIntegration returns the one integration that matches every filter that
// is set. The API filters by name and team; type is filtered here.
func findIntegration(client *onCallAPI.Client, name, teamID, integrationType string) (*onCallAPI.Integration, diag.Diagnostics) {
	options := &onCallAPI.ListIntegrationOptions{
		Name:   name,
		TeamID: teamID,
	}

	var matches []*onCallAPI.Integration
	for page := 1; ; page++ {
		options.Page = page
		integrationsResponse, _, err := client.Integrations.ListIntegrations(options)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		for _, integration := range integrationsResponse.Integrations {
			if integrationType == "" || integration.Type == integrationType {
				matches = append(matches, integration)
			}
		}
		if integrationsResponse.Next == nil {
			break
		}
	}

	switch len(matches) {
	case 0:
		return nil, diag.Errorf("couldn't find an integration matching %s", describeIntegrationFilters(name, teamID, integrationType))
	case 1:
		return matches[0], nil
	default:
		return nil, diag.Errorf("%d integrations match %s; set more of name, team_id and type to match exactly one", len(matches), describeIntegrationFilters(name, teamID, integrationType))
	}
}

func describeIntegrationFilters(name, teamID, integrationType string) string {
	var filters []string
	if name != "" {
		filters = append(filters, fmt.Sprintf("name %q", name))
	}
	if teamID != "" {
		filters = append(filters, fmt.Sprintf("team_id %q", teamID))
	}
	if integrationType != "" {
		filters = append(filters, fmt.Sprintf("type %q", integrationType))
	}
	return strings.Join(filters, ", ")
}
