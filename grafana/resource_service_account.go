package grafana

import (
	"context"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceServiceAccount() *schema.Resource {
	return &schema.Resource{

		Description: `

This resource uses Grafana's API for creating and updating service accounts.`,

		CreateContext: CreateServiceAccount,
		ReadContext:   ReadServiceAccount,
		UpdateContext: UpdateServiceAccount,
		DeleteContext: DeleteServiceAccount,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The numerical ID of the Grafana user.",
			},
			"login": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The username of the service account",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name for the Grafana service account.",
			},
			"role": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The role of the service account in the organization.",
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "The disabled status for the service account.",
			},
		},
	}
}

func CreateServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	req := gapi.CreateServiceAccountRequest{
		Name: d.Get("name").(string),
	}
	sa, err := client.CreateServiceAccount(req)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("role") || d.HasChange("is_disabled") {
		updateRequest := gapi.UpdateServiceAccountRequest{
			Name: req.Name,
		}
		if d.HasChange("role") {
			updateRequest.Role = d.Get("role").(string)
		}
		if d.HasChange("is_disabled") {
			isDisabled := d.Get("is_disabled").(bool)
			updateRequest.IsDisabled = &isDisabled
		}

		if _, err := client.UpdateServiceAccount(sa.ID, updateRequest); err != nil {
			return diag.FromErr(err)
		}
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
			d.Set("name", sa.Name)
			d.Set("login", sa.Login)
			d.Set("role", sa.Role)
			d.Set("is_disabled", sa.IsDisabled)

			return nil
		}
	}
	// Resource was not found via the client. Have Terraform destroy it.
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
		isDisabled := d.Get("is_disabled").(bool)
		updateRequest.IsDisabled = &isDisabled
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
