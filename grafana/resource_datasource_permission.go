package grafana

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceDatasourcePermission() *schema.Resource {
	return &schema.Resource{

		Description: `
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/datasource_permissions/)
`,

		CreateContext: UpdateDatasourcePermissions,
		ReadContext:   ReadDatasourcePermissions,
		UpdateContext: UpdateDatasourcePermissions,
		DeleteContext: DeleteDatasourcePermissions,

		Schema: map[string]*schema.Schema{
			"datasource_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the datasource to apply permissions to.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "The permission items to add/update. Items that are omitted from the list will be removed.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"team_id": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "ID of the team to manage permissions for.",
						},
						"user_id": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "ID of the user to manage permissions for.",
						},
						"built_in_role": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "",
							ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor"}, false),
							Description:  "Name of the basic role to manage permissions for. Options: `Viewer` or `Editor`.",
						},
						"permission": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"Query"}, false),
							Description:  "Permission to associate with item. Must be `Query`.",
						},
					},
				},
			},
		},
	}
}

func UpdateDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	v, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}
	datasourceID := int64(d.Get("datasource_id").(int))

	configuredPermissions := []*gapi.DatasourcePermissionAddPayload{}
	for _, permission := range v.(*schema.Set).List() {
		permission := permission.(map[string]interface{})
		permissionItem := gapi.DatasourcePermissionAddPayload{}
		if permission["team_id"].(int) != -1 {
			permissionItem.TeamID = int64(permission["team_id"].(int))
		}
		if permission["user_id"].(int) != -1 {
			permissionItem.UserID = int64(permission["user_id"].(int))
		}
		if permission["built_in_role"].(string) != "" {
			permissionItem.BuiltInRole = permission["built_in_role"].(string)
		}
		var err error
		if permissionItem.Permission, err = mapDatasourcePermissionStringToType(permission["permission"].(string)); err != nil {
			return diag.FromErr(err)
		}
		configuredPermissions = append(configuredPermissions, &permissionItem)
	}

	if err := updateDatasourcePermissions(client, datasourceID, configuredPermissions, true, false); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(datasourceID, 10))

	return ReadDatasourcePermissions(ctx, d, meta)
}

func ReadDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	response, err := client.DatasourcePermissions(id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing datasource permissions %d from state because it no longer exists in grafana", id)
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	permissionItems := make([]interface{}, len(response.Permissions))
	for _, permission := range response.Permissions {
		permissionItem := make(map[string]interface{})
		// Cannot change Admin role permissions on data sources, so ignore it when provisioning
		if permission.BuiltInRole == "Admin" {
			continue
		}
		permissionItem["built_in_role"] = permission.BuiltInRole
		permissionItem["team_id"] = permission.TeamID
		permissionItem["user_id"] = permission.UserID

		if permissionItem["permission"], err = mapDatasourcePermissionTypeToString(permission.Permission); err != nil {
			return diag.FromErr(err)
		}

		permissionItems = append(permissionItems, permissionItem)
	}

	d.Set("permissions", permissionItems)

	return nil
}

func DeleteDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	datasourceID := int64(d.Get("datasource_id").(int))

	if err := updateDatasourcePermissions(client, datasourceID, []*gapi.DatasourcePermissionAddPayload{}, false, true); err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing datasource permissions %d from state because it no longer exists in grafana", datasourceID)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func updateDatasourcePermissions(client *gapi.Client, id int64, permissions []*gapi.DatasourcePermissionAddPayload, enable, disable bool) error {
	areEqual := func(a *gapi.DatasourcePermission, b *gapi.DatasourcePermissionAddPayload) bool {
		return a.Permission == b.Permission && a.TeamID == b.TeamID && a.UserID == b.UserID && a.BuiltInRole == b.BuiltInRole
	}

	response, err := client.DatasourcePermissions(id)
	if err != nil {
		return err
	}

	if !response.Enabled && enable {
		if err := client.EnableDatasourcePermissions(id); err != nil {
			return err
		}
	}

deleteLoop:
	for _, current := range response.Permissions {
		for _, new := range permissions {
			if areEqual(current, new) {
				continue deleteLoop
			}
		}

		err := client.RemoveDatasourcePermission(id, current.ID)
		if err != nil {
			return err
		}
	}

addLoop:
	for _, new := range permissions {
		for _, current := range response.Permissions {
			if areEqual(current, new) {
				continue addLoop
			}
		}

		err := client.AddDatasourcePermission(id, new)
		if err != nil {
			return err
		}
	}

	if response.Enabled && disable {
		if err := client.DisableDatasourcePermissions(id); err != nil {
			return err
		}
	}

	return nil
}

func mapDatasourcePermissionStringToType(permission string) (gapi.DatasourcePermissionType, error) {
	if permission == "Query" {
		return gapi.DatasourcePermissionQuery, nil
	}
	return 0, fmt.Errorf("unknown datasource permission: %s", permission)
}

func mapDatasourcePermissionTypeToString(permission gapi.DatasourcePermissionType) (string, error) {
	if permission == gapi.DatasourcePermissionQuery {
		return "Query", nil
	}
	return "", fmt.Errorf("unknown permission type: %d", permission)
}
