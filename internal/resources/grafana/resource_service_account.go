package grafana

import (
	"context"
	"log"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

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
				ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin"}, false),
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
	client, orgID := ClientFromNewOrgResource(meta, d)
	isDisabled := d.Get("is_disabled").(bool)
	req := gapi.CreateServiceAccountRequest{
		Name:       d.Get("name").(string),
		Role:       d.Get("role").(string),
		IsDisabled: &isDisabled,
	}
	sa, err := client.CreateServiceAccount(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, sa.ID))
	return ReadServiceAccount(ctx, d, meta)
}

func ReadServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	sas, err := client.GetServiceAccounts()
	if err != nil {
		return diag.FromErr(err)
	}

	for _, sa := range sas {
		if sa.ID == id {
			d.SetId(MakeOrgResourceID(sa.OrgID, id))
			d.Set("org_id", strconv.FormatInt(sa.OrgID, 10))
			err = d.Set("name", sa.Name)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set("role", sa.Role)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set("is_disabled", sa.IsDisabled)
			if err != nil {
				return diag.FromErr(err)
			}

			return nil
		}
	}
	log.Printf("[WARN] removing service account %d from state because it no longer exists in grafana", id)
	d.SetId("")

	return nil
}

func UpdateServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := gapi.UpdateServiceAccountRequest{}
	if d.HasChange("name") {
		updateRequest.Name = d.Get("name").(string)
	}
	if d.HasChange("role") {
		updateRequest.Role = d.Get("role").(string)
	}
	if d.HasChange("is_disabled") {
		isDisabled := d.Get("is_disabled").(bool)
		updateRequest.IsDisabled = &isDisabled
	}

	if _, err := client.UpdateServiceAccount(id, updateRequest); err != nil {
		return diag.FromErr(err)
	}

	return ReadServiceAccount(ctx, d, meta)
}

func DeleteServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.DeleteServiceAccount(id)
	return diag.FromErr(err)
}
