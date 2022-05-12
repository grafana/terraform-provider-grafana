package grafana

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceOrganization() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/manage-organizations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/org/)
`,
		ReadContext: dataSourceOrganizationRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Organization.",
			},
		},
	}
}

func dataSourceOrganizationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	name := d.Get("name").(string)
	org, err := client.OrgByName(name)

	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(org.ID, 10))
	return nil
}
