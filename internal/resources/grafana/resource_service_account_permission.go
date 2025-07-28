package grafana

import (
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceServiceAccountPermission() *common.Resource {
	crudHelper := &resourcePermissionsHelper{
		resourceType: serviceAccountsPermissionsType,
		getResource:  resourceServiceAccountPermissionGet,
	}

	schema := &schema.Resource{
		Description: `
Manages the entire set of permissions for a service account. Permissions that aren't specified when applying this resource will be removed.

**Note:** This resource is available from Grafana 9.2.4 onwards.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/#manage-users-and-teams-permissions-for-a-service-account-in-grafana)`,

		CreateContext: crudHelper.updatePermissions,
		ReadContext:   crudHelper.readPermissions,
		UpdateContext: crudHelper.updatePermissions,
		DeleteContext: crudHelper.deletePermissions,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"service_account_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the service account.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, old = SplitServiceAccountID(old)
					_, new = SplitServiceAccountID(new)
					return old == new
				},
			},
		},
	}
	crudHelper.addCommonSchemaAttributes(schema.Schema)

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_service_account_permission",
		orgResourceIDInt("serviceAccountID"),
		schema,
	)
}

func resourceServiceAccountPermissionGet(d *schema.ResourceData, meta interface{}) (string, error) {
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	_, id := SplitServiceAccountID(d.Get("service_account_id").(string))
	if d.Id() != "" {
		client, _, id = OAPIClientFromExistingOrgResource(meta, d.Id())
	}
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", err
	}
	resp, err := client.ServiceAccounts.RetrieveServiceAccount(idInt)
	if err != nil {
		return "", err
	}
	sa := resp.Payload
	id = strconv.FormatInt(sa.ID, 10)
	d.Set("service_account_id", id)
	return id, nil
}
