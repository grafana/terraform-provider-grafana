package grafana

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
)

func resourceDashboardPermission() *common.Resource {
	crudHelper := &resourcePermissionsHelper{
		resourceType:  dashboardsPermissionsType,
		roleAttribute: "role",
		getResource:   resourceDashboardPermissionGet,
	}

	schema := &schema.Resource{
		Description: `
Manages the entire set of permissions for a dashboard. Permissions that aren't specified when applying this resource will be removed.
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard_permissions/)
`,

		CreateContext: crudHelper.updatePermissions,
		ReadContext:   crudHelper.readPermissions,
		UpdateContext: crudHelper.updatePermissions,
		DeleteContext: crudHelper.deletePermissions,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"dashboard_id": {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{"dashboard_id", "dashboard_uid"},
				Deprecated:   "use `dashboard_uid` instead",
				Description:  "ID of the dashboard to apply permissions to. Deprecated: use `dashboard_uid` instead.",
			},
			"dashboard_uid": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{"dashboard_id", "dashboard_uid"},
				Description:  "UID of the dashboard to apply permissions to.",
			},
		},
	}
	crudHelper.addCommonSchemaAttributes(schema.Schema)

	return common.NewLegacySDKResource(
		"grafana_dashboard_permission",
		orgResourceIDString("dashboardUID"),
		schema,
	)
}

func resourceDashboardPermissionGet(d *schema.ResourceData, meta interface{}) (string, error) {
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	uid := d.Get("dashboard_uid").(string)
	if d.Id() != "" {
		client, _, uid = OAPIClientFromExistingOrgResource(meta, d.Id())
	}

	if uid != "" {
		_, err := client.Dashboards.GetDashboardByUID(uid)
		if err != nil {
			return "", err
		}
		d.Set("dashboard_uid", uid)
		return uid, nil
	}
	id := int64(d.Get("dashboard_id").(int))
	dashboard, err := getDashboardByID(client, id)
	if err != nil {
		return "", err
	}
	d.Set("dashboard_uid", dashboard.UID)
	return dashboard.UID, nil
}
