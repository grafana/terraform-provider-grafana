package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/grafana/terraform-provider-grafana/provider/common"
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
			"admins": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "A list of email addresses corresponding to users given admin access to the organization.",
			},
			"editors": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "A list of email addresses corresponding to users given editor access to the organization.",
			},
			"viewers": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "A list of email addresses corresponding to users given viewer access to the organization.",
			},
		},
	}
}

func dataSourceOrganizationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	name := d.Get("name").(string)
	org, err := client.OrgByName(name)

	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			return diag.Errorf("no organization with name %q", name)
		}
		return diag.FromErr(err)
	}

	orgUsers, err := client.OrgUsers(org.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	userCollections := map[string][]string{
		"admins":  []string{},
		"editors": []string{},
		"viewers": []string{},
	}

	for _, user := range orgUsers {
		role := fmt.Sprintf("%ss", strings.ToLower(user.Role))
		userCollections[role] = append(userCollections[role], user.Email)
	}

	for role, emails := range userCollections {
		if err := d.Set(role, emails); err != nil {
			return diag.Errorf("error setting %s: %v", role, err)
		}
	}

	d.SetId(strconv.FormatInt(org.ID, 10))
	return nil
}
