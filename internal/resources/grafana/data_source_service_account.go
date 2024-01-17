package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		Description: `
		* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
		* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)
`,
		ReadContext: DatasourceServiceAccountRead,
		Schema: common.CloneResourceSchemaForDatasource(ResourceServiceAccount(), map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Service Account.",
			},
		}),
	}
}

func DatasourceServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	name := d.Get("name").(string)
	var page int64 = 0
	for {
		params := service_accounts.NewSearchOrgServiceAccountsWithPagingParams().WithPage(&page)
		resp, err := client.ServiceAccounts.SearchOrgServiceAccountsWithPaging(params)
		if err != nil {
			return diag.FromErr(err)
		}
		serviceAccounts := resp.Payload.ServiceAccounts
		if len(serviceAccounts) == 0 {
			break
		}
		for _, sa := range serviceAccounts {
			if sa.Name == name {
				d.SetId(MakeOrgResourceID(orgID, sa.ID))
				return ReadServiceAccount(ctx, d, meta)
			}
		}
	}
	return diag.Errorf("service account %q not found", name)
}
