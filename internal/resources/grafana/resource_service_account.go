package grafana

import (
	"context"
	"strconv"
	"sync"

	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Service Accounts have issues with concurrent creation, so we need to lock them.
var serviceAccountCreateMutex sync.Mutex

func ResourceServiceAccount() *schema.Resource {
	return &schema.Resource{

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
				Optional:     true,
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
}

func CreateServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	serviceAccountCreateMutex.Lock()
	defer serviceAccountCreateMutex.Unlock()

	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	req := models.CreateServiceAccountForm{
		Name:       d.Get("name").(string),
		Role:       d.Get("role").(string),
		IsDisabled: d.Get("is_disabled").(bool),
	}

	params := service_accounts.NewCreateServiceAccountParams().WithBody(&req)
	resp, err := client.ServiceAccounts.CreateServiceAccount(params, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, resp.Payload.ID))

	return ReadServiceAccount(ctx, d, meta)
}

func ReadServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	params := service_accounts.NewRetrieveServiceAccountParams().WithServiceAccountID(id)
	resp, err := client.ServiceAccounts.RetrieveServiceAccount(params, nil)
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

func UpdateServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := models.UpdateServiceAccountForm{}
	if d.HasChange("name") {
		updateRequest.Name = d.Get("name").(string)
	}
	if d.HasChange("role") {
		updateRequest.Role = d.Get("role").(string)
	}
	if d.HasChange("is_disabled") {
		updateRequest.IsDisabled = d.Get("is_disabled").(bool)
	}

	params := service_accounts.NewUpdateServiceAccountParams().
		WithBody(&updateRequest).
		WithServiceAccountID(id)
	if _, err := client.ServiceAccounts.UpdateServiceAccount(params, nil); err != nil {
		return diag.FromErr(err)
	}

	return ReadServiceAccount(ctx, d, meta)
}

func DeleteServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	params := service_accounts.NewDeleteServiceAccountParams().WithServiceAccountID(id)
	_, err = client.ServiceAccounts.DeleteServiceAccount(params, nil)
	diag, _ := common.CheckReadError("service account", d, err)
	return diag
}
