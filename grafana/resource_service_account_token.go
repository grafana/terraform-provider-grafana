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
Manages Grafana Service Account Tokens.

* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/serviceaccount/)
`,

		CreateContext: resourceServiceAccountTokenCreate,
		ReadContext:   resourceServiceAccountTokenRead,
		DeleteContext: resourceServiceAccountTokenDelete,

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

			"id": {
				Type:     schema.TypeString,
				Computed: true,
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

func resourceServiceAccountTokenCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	serviceAccountID := d.Get("service_account_id").(int)
	ttl := d.Get("seconds_to_live").(int)
	c := m.(*client).gapi

	request := gapi.CreateServiceAccountTokenRequest{
		Name:             name,
		ServiceAccountID: int64(serviceAccountID),
		SecondsToLive:    int64(ttl)}
	response, err := c.CreateServiceAccountToken(request)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(response.ID, 10))
	d.Set("key", response.Key)

	// Fill the true resource's state after a create by performing a read
	return resourceServiceAccountTokenRead(ctx, d, m)
}

func resourceServiceAccountTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceAccountID := d.Get("service_account_id").(int64)
	c := m.(*client).gapi

	response, err := c.GetServiceAccountTokens(serviceAccountID)
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
			d.Set("name", key.Name)

			if !key.Expiration.IsZero() {
				d.Set("expiration", key.Expiration.String())
			}

			return nil
		}
	}

	// Resource was not found via the client. Have Terraform destroy it.
	d.SetId("")

	return nil
}

func resourceServiceAccountTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceAccountID := d.Get("service_account_id").(int64)
	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	c := m.(*client).gapi

	_, err = c.DeleteServiceAccountToken(serviceAccountID, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
