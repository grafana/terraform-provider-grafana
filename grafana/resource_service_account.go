package grafana

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceServiceAccount() *schema.Resource {
	return &schema.Resource{

		Description: `
**Note:** This resource is available only with Grafana 9.0+.

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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
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
	client := meta.(*client).gapi
	req := gapi.CreateServiceAccountRequest{
		Name:       d.Get("name").(string),
		Role:       d.Get("role").(string),
		IsDisabled: d.Get("is_disabled").(bool),
	}
	sa, err := client.CreateServiceAccount(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(sa.ID, 10))
	return ReadServiceAccount(ctx, d, meta)
}

func ReadServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	sas, err := client.GetServiceAccounts()
	if err != nil {
		return diag.FromErr(err)
	}

	for _, sa := range sas {
		if sa.ID == id {
			d.SetId(strconv.FormatInt(sa.ID, 10))
			err = d.Set("name", sa.Name)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set("login", sa.Login)
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
	// Resource was not found via the client. Enforce Terraform to destroy it.
	d.SetId("")

	return nil
}

func UpdateServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := gapi.UpdateServiceAccountRequest{}
	if d.HasChange("role") {
		updateRequest.Role = d.Get("role").(string)
	}
	if d.HasChange("is_disabled") {
		updateRequest.IsDisabled = d.Get("is_disabled").(bool)
	}

	if _, err := client.UpdateServiceAccount(id, updateRequest); err != nil {
		return diag.FromErr(err)
	}

	return ReadServiceAccount(ctx, d, meta)
}

func DeleteServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err = client.DeleteServiceAccount(id); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}
