package grafana

import (
	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// datasourcePermissionListClientOptsFromState supplies GetResourcePermissions query params from config/state only
// (ds_type when datasource_type is set — no datasource API call).
func datasourcePermissionListClientOptsFromState(d *schema.ResourceData) []access_control.ClientOption {
	if t, ok := d.GetOk("datasource_type"); ok && t.(string) != "" {
		return []access_control.ClientOption{withQueryParam("ds_type", t.(string))}
	}
	return nil
}

func resourceDatasourcePermission() *common.Resource {
	crudHelper := &resourcePermissionsHelper{
		resourceType:              datasourcesPermissionsType,
		roleAttribute:             "built_in_role",
		getResource:               resourceDatasourcePermissionGet,
		listPermissionsClientOpts: datasourcePermissionListClientOptsFromState,
		buildClientOptions: func(d *schema.ResourceData, meta any) ([]access_control.ClientOption, error) {
			dsType, err := resourceDatasourcePermissionGetType(d, meta)
			if err != nil {
				return nil, err
			}
			if dsType == "" {
				return nil, nil
			}
			return []access_control.ClientOption{withQueryParam("ds_type", dsType)}, nil
		},
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
			"datasource_type": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The plugin type of the datasource (e.g. \"prometheus\"). If set, skips the lookup of the datasource type from the API.",
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

func resourceDatasourcePermissionGet(d *schema.ResourceData, meta any) (string, error) {
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

func resourceDatasourcePermissionGetType(d *schema.ResourceData, meta any) (string, error) {
	if t, ok := d.GetOk("datasource_type"); ok {
		return t.(string), nil
	}
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	_, id := SplitOrgResourceID(d.Get("datasource_uid").(string))
	if d.Id() != "" {
		client, _, id = OAPIClientFromExistingOrgResource(meta, d.Id())
	}
	resp, err := client.Datasources.GetDataSourceByUID(id)
	if err != nil {
		return "", err
	}
	return resp.Payload.Type, nil
}
