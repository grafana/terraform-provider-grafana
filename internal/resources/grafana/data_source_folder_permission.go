package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceFolderPermission() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/manage-dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_permissions/)
`,
		ReadContext: dataSourceFolderPermissionRead,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"folder_uid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The UID of the folder.",
			},
			"permissions": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"team_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the team to manage permissions for.",
						},
						"user_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the user or service account to manage permissions for.",
						},
						"permission": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Permission to associate with item. Must be one of `View`, `Edit`, or `Admin`.",
						},
						"role": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Role to associate with item. Must be one of `Viewer`, `Editor`, or `Admin`.",
						},
					},
				},
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_folder_permission", schema)
}

func dataSourceFolderPermissionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	uid := d.Get("folder_uid").(string)

	resp, err := client.FolderPermissions.GetFolderPermissionList(uid)
	if err != nil {
		return diag.FromErr(err)
	}

	var resourcePermissions []models.DashboardACLInfoDTO
	for _, perm := range resp.Payload {
		resourcePermissions = append(resourcePermissions, *perm)
	}

	var permissionItems []interface{}
	for _, permission := range resourcePermissions {
		permissionItem := make(map[string]interface{})
		if permission.Role != "" {
			permissionItem["role"] = permission.Role
		}
		permissionItem["team_id"] = permission.TeamUID
		permissionItem["user_id"] = permission.UserUID
		permissionItem["permission"] = permission.PermissionName

		permissionItems = append(permissionItems, permissionItem)
	}

	d.SetId(MakeOrgResourceID(orgID, uid))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("permissions", permissionItems)

	return nil
}
