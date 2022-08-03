package grafana

import (
	"context"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceServiceAccountToken() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana 9.1+.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)`,

		CreateContext: serviceAccountTokenCreate,
		ReadContext:   serviceAccountTokenRead,
		DeleteContext: serviceAccountTokenDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"seconds_to_live": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"expiration": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"has_expired": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func serviceAccountTokenCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	serviceAccountID := d.Get("service_account_id").(int)
	ttl := d.Get("seconds_to_live").(int)

	c := m.(*client).gapi

	request := gapi.CreateServiceAccountTokenRequest{
		Name:             name,
		ServiceAccountID: int64(serviceAccountID),
		SecondsToLive:    int64(ttl),
	}
	response, err := c.CreateServiceAccountToken(request)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(response.ID, 10))
	err = d.Set("key", response.Key)
	if err != nil {
		return diag.FromErr(err)
	}

	// Fill the true resource's state by performing a read
	return serviceAccountTokenRead(ctx, d, m)
}

func serviceAccountTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceAccountID := d.Get("service_account_id").(int)
	c := m.(*client).gapi

	response, err := c.GetServiceAccountTokens(int64(serviceAccountID))
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, key := range response {
		if id == key.ID {
			d.SetId(strconv.FormatInt(key.ID, 10))
			err = d.Set("name", key.Name)
			if err != nil {
				return diag.FromErr(err)
			}
			if key.Expiration != nil && !key.Expiration.IsZero() {
				err = d.Set("expiration", key.Expiration.String())
				if err != nil {
					return diag.FromErr(err)
				}
			}
			err = d.Set("has_expired", key.HasExpired)

			return diag.FromErr(err)
		}
	}

	// Resource was not found via the client. Enforce Terraform destroy it.
	d.SetId("")

	return nil
}

func serviceAccountTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceAccountID := d.Get("service_account_id").(int)
	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	c := m.(*client).gapi

	_, err = c.DeleteServiceAccountToken(int64(serviceAccountID), id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
