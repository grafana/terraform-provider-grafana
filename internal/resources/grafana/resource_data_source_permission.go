package grafana

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
)

func resourceDatasourcePermission() *common.Resource {
	crudHelper := &resourcePermissionsHelper{
		resourceType:  datasourcesPermissionsType,
		roleAttribute: "built_in_role",
		getResource:   resourceDatasourcePermissionGet,
	}

	schema := &schema.Resource{
		Description: `
Manages the entire set of permissions for a datasource. Permissions that aren't specified when applying this resource will be removed.
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/datasource_permissions/)
`,

		CreateContext: crudHelper.updatePermissions,
		ReadContext:   crudHelper.readPermissions,
		UpdateContext: crudHelper.updatePermissions,
		DeleteContext: crudHelper.deletePermissions,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"datasource_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				Deprecated:   "Use `datasource_uid` instead",
				Description:  "Deprecated: Use `datasource_uid` instead.",
				AtLeastOneOf: []string{"datasource_id", "datasource_uid"},
			},
			"datasource_uid": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				Description:  "UID of the datasource to apply permissions to.",
				AtLeastOneOf: []string{"datasource_id", "datasource_uid"},
			},
		},
	}
	crudHelper.addCommonSchemaAttributes(schema.Schema)

	return common.NewLegacySDKResource(
		"grafana_data_source_permission",
		orgResourceIDInt("datasourceID"),
		schema,
	)
}

func resourceDatasourcePermissionGet(d *schema.ResourceData, meta interface{}) (string, error) {
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	_, id := SplitOrgResourceID(d.Get("datasource_uid").(string))
	if d.Id() != "" {
		client, _, id = OAPIClientFromExistingOrgResource(meta, d.Id())
	}
	if id == "" {
		_, id = SplitOrgResourceID(d.Get("datasource_id").(string))
	}
	datasource, err := getDatasourceByUIDOrID(client, id)
	if err != nil {
		return "", err
	}
	d.Set("datasource_uid", datasource.UID)
	return datasource.UID, nil
}
