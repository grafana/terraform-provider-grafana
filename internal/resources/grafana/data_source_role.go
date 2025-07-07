package grafana

import (
	"context"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceRole() *common.DataSource {
	schema := &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 8.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/access_control/)
`,
		ReadContext: dataSourceRoleRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceRole().Schema, map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the role",
				ValidateFunc: func(i interface{}, k string) (warnings []string, errors []error) {
					name := i.(string)
					if strings.HasPrefix(strings.ToLower(name), "plugins:grafana-oncall-app:") {
						warnings = append(warnings, "Roles from 'grafana-oncall-app' are deprecated and should be migrated to 'grafana-irm-app' roles instead.")
					}
					return warnings, nil
				},
			},
			"auto_increment_version": nil,
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaEnterprise, "grafana_role", schema)
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	resp, err := client.AccessControl.ListRoles(access_control.NewListRolesParams().WithIncludeHidden(common.Ref(true)), nil)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	for _, r := range resp.Payload {
		if r.Name == name {
			d.SetId(MakeOrgResourceID(orgID, r.UID))
			return readRoleFromUID(client, r.UID, d)
		}
	}

	return diag.Errorf("no role with name %q", name)
}
