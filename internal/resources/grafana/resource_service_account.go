package grafana

import (
	"context"
	"strconv"
	"sync"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Service Accounts have issues with concurrent creation, so we need to lock them.
var serviceAccountCreateMutex sync.Mutex

func resourceServiceAccount() *common.Resource {
	schema := &schema.Resource{

		Description: `
**Note:** This resource is available only with Grafana 9.1+.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)`,

		CreateContext: CreateServiceAccount,
		ReadContext:   ReadServiceAccount,
		UpdateContext: UpdateServiceAccount,
		DeleteContext: DeleteServiceAccount,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the service account.",
			},
			"role": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin", "None"}, false),
				Description:  "The basic role of the service account in the organization.",
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The disabled status for the service account.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_service_account",
		orgResourceIDInt("id"),
		schema,
	).
		WithLister(listerFunctionOrgResource(listServiceAccounts)).
		WithPreferredResourceNameField("name")
}

func listServiceAccounts(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	var page int64 = 1
	for {
		params := service_accounts.NewSearchOrgServiceAccountsWithPagingParams().WithPage(&page)
		resp, err := client.ServiceAccounts.SearchOrgServiceAccountsWithPaging(params)
		if err != nil {
			return nil, err
		}

		for _, sa := range resp.Payload.ServiceAccounts {
			ids = append(ids, MakeOrgResourceID(orgID, sa.ID))
		}

		if resp.Payload.TotalCount <= int64(len(ids)) {
			break
		}

		page++
	}

	return ids, nil
}

func CreateServiceAccount(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	serviceAccountCreateMutex.Lock()
	defer serviceAccountCreateMutex.Unlock()

	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	client = client.WithRetries(0, 0) // Disable retries to have our own retry logic
	req := models.CreateServiceAccountForm{
		Name:       d.Get("name").(string),
		Role:       d.Get("role").(string),
		IsDisabled: d.Get("is_disabled").(bool),
	}

	var sa *models.ServiceAccountDTO
	err := retry.RetryContext(ctx, 10*time.Second, func() *retry.RetryError {
		params := service_accounts.NewCreateServiceAccountParams().WithBody(&req)
		resp, err := client.ServiceAccounts.CreateServiceAccount(params)
		if err == nil {
			sa = resp.Payload
			return nil
		}

		if err, ok := err.(*service_accounts.CreateServiceAccountInternalServerError); ok {
			// Sometimes on 500s, the service account is created but the response is not returned.
			// If we just retry, it will conflict because the SA was actually created.
			foundSa, readErr := findServiceAccountByName(client, req.Name)
			if readErr != nil {
				return retry.RetryableError(err)
			}
			sa = foundSa
			return nil
		}
		return retry.NonRetryableError(err)
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, sa.ID))

	return ReadServiceAccount(ctx, d, meta)
}

func ReadServiceAccount(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.ServiceAccounts.RetrieveServiceAccount(id)
	if err, shouldReturn := common.CheckReadError("service account", d, err); shouldReturn {
		return err
	}
	sa := resp.Payload

	d.SetId(MakeOrgResourceID(sa.OrgID, id))
	d.Set("org_id", strconv.FormatInt(sa.OrgID, 10))
	d.Set("name", sa.Name)
	d.Set("role", sa.Role)
	d.Set("is_disabled", sa.IsDisabled)
	return nil
}

func UpdateServiceAccount(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := models.UpdateServiceAccountForm{
		Name:       d.Get("name").(string),
		Role:       d.Get("role").(string),
		IsDisabled: d.Get("is_disabled").(bool),
	}

	params := service_accounts.NewUpdateServiceAccountParams().
		WithBody(&updateRequest).
		WithServiceAccountID(id)
	if _, err := client.ServiceAccounts.UpdateServiceAccount(params); err != nil {
		return diag.FromErr(err)
	}

	return ReadServiceAccount(ctx, d, meta)
}

func DeleteServiceAccount(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.ServiceAccounts.DeleteServiceAccount(id)
	diag, _ := common.CheckReadError("service account", d, err)
	return diag
}
