package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceRole() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 8.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/access_control/)
`,
		ReadContext: dataSourceRoleRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceRole(), map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the role",
			},
			"auto_increment_version": nil,
		}),
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	resp, err := client.AccessControl.ListRoles(access_control.NewListRolesParams(), nil)
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
