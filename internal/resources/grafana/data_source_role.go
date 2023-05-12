package grafana

import (
	"context"
	"fmt"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceRole() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 8.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/access_control/)
`,
		ReadContext: dataSourceRoleRead,
		Schema: map[string]*schema.Schema{
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier of the role. Used for assignments.",
			},
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Version of the role. A role is updated only on version increase.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the role",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the role.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Display name of the role. Available with Grafana 8.5+.",
			},
			"group": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Group of the role. Available with Grafana 8.5+.",
			},
			"hidden": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Boolean to state whether the role should be visible in the Grafana UI or not. Available with Grafana 8.5+.",
			},
			"global": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Boolean to state whether the role is available across all organizations or not.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Specific set of actions granted by the role.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Specific action users granted with the role will be allowed to perform (for example: `users:read`)",
						},
						"scope": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Scope to restrict the action to a set of resources (for example: `users:*` or `roles:customrole1`)",
						},
					},
				},
			},
		},
	}
}

func findRoleWithName(client *gapi.Client, name string) (*gapi.Role, error) {
	roles, err := client.GetRoles()
	if err != nil {
		return nil, err
	}

	for _, r := range roles {
		if r.Name == name {
			// Query the role by UID, that API has additional information
			return client.GetRole(r.UID)
		}
	}

	return nil, fmt.Errorf("no role with name %q", name)
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	name := d.Get("name").(string)
	role, err := findRoleWithName(client, name)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(role.UID)
	d.Set("uid", role.UID)
	d.Set("version", role.Version)
	d.Set("name", role.Name)
	if role.Description != "" {
		d.Set("description", role.Description)
	}
	if role.DisplayName != "" {
		d.Set("displayName", role.DisplayName)
	}
	if role.Group != "" {
		d.Set("group", role.Group)
	}
	d.Set("hidden", role.Hidden)
	d.Set("global", role.Global)
	perms := make([]interface{}, 0)
	for _, p := range role.Permissions {
		pMap := map[string]interface{}{
			"action": p.Action,
			"scope":  p.Scope,
		}
		perms = append(perms, pMap)
	}
	err = d.Set("permissions", perms)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
