package grafana

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceServiceAccount() *common.DataSource {
	schema := &schema.Resource{
		Description: `
		* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
		* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)
`,
		ReadContext: datasourceServiceAccountRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceServiceAccount().Schema, map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Service Account.",
			},
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_service_account", schema)
}

func datasourceServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	name := d.Get("name").(string)
	sa, err := findServiceAccountByName(client, name)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, sa.ID))
	return ReadServiceAccount(ctx, d, meta)
}

func findServiceAccountByName(client *client.GrafanaHTTPAPI, name string) (*models.ServiceAccountDTO, error) {
	var page int64 = 0
	for {
		params := service_accounts.NewSearchOrgServiceAccountsWithPagingParams().WithPage(&page)
		resp, err := client.ServiceAccounts.SearchOrgServiceAccountsWithPaging(params)
		if err != nil {
			return nil, err
		}
		serviceAccounts := resp.Payload.ServiceAccounts
		if len(serviceAccounts) == 0 {
			break
		}
		for _, sa := range serviceAccounts {
			if sa.Name == name {
				return sa, nil
			}
		}
	}
	return nil, fmt.Errorf("service account %q not found", name)
}
