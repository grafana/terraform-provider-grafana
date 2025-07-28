package grafana

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
			"datasource_uid": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "UID of the datasource to apply permissions to.",
			},
		},
	}
	crudHelper.addCommonSchemaAttributes(schema.Schema)

	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
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
	resp, err := client.Datasources.GetDataSourceByUID(id)
	if err != nil {
		return "", err
	}
	datasource := resp.Payload
	d.Set("datasource_uid", datasource.UID)
	return datasource.UID, nil
}
