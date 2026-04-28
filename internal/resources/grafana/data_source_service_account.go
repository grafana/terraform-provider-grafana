package grafana

import (
	"context"
	"fmt"
	"strconv"

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
		* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/serviceaccount/#service-account-api)
`,
		ReadContext: datasourceServiceAccountRead,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Service Account.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of this resource.",
			},
			"role": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The basic role of the service account in the organization.",
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "The disabled status for the service account.",
			},
		},
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
	d.SetId(MakeOrgResourceID(orgID, strconv.FormatInt(sa.ID, 10)))
	d.Set("org_id", strconv.FormatInt(sa.OrgID, 10))
	d.Set("name", sa.Name)
	d.Set("role", sa.Role)
	d.Set("is_disabled", sa.IsDisabled)
	return nil
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
		page++
	}
	return nil, fmt.Errorf("service account %q not found", name)
}
