package grafana

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceOrganizationUsers() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/manage-organizations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/org/)
`,
		ReadContext: dataSourceOrganizationUsersRead,
		Schema: map[string]*schema.Schema{
			"organization_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Organization.",
			},
			"emails": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of the Organization member users' email addresses.",
			},
		},
	}
}

func dataSourceOrganizationUsersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	name := d.Get("organization_name").(string)
	org, err := client.OrgByName(name)
	if err != nil {
		return diag.FromErr(err)
	}

	orgUsers, err := client.OrgUsers(org.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	var collectedEmails []string
	for _, user := range orgUsers {
		collectedEmails = append(collectedEmails, user.Email)
	}

	attr := "emails"
	if err := d.Set(attr, collectedEmails); err != nil {
		return diag.FromErr(fmt.Errorf("error setting %s: %v", attr, err))
	}

	d.SetId(strconv.FormatInt(org.ID, 10))
	return nil
}
