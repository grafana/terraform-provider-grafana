package grafana

import (
	"context"
	"strconv"
	"time"

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

		CreateContext: resourceAPIKeyCreate,
		ReadContext:   resourceAPIKeyRead,
		DeleteContext: resourceAPIKeyDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account_id": {
				Type:     schema.TypeInt,
				ForceNew: true,
			},
			"seconds_to_live": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"id": {
				Type:     schema.TypeInt,
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
	role := d.Get("service_account_id").(int)
	ttl := d.Get("seconds_to_live").(int)
	c := m.(*client).gapi

	request := gapi.CreateAPIKeyRequest{Name: name, Role: role, SecondsToLive: int64(ttl)}
	response, err := c.CreateAPIKey(request)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(response.ID, 10))
	d.Set("key", response.Key)

	// Fill the true resource's state after a create by performing a read
	return resourceAPIKeyRead(ctx, d, m)
}

func resourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, cleanup, err := getClientForAPIKeyManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	response, err := c.GetAPIKeys(true)
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
			d.Set("role", key.Role)

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

func resourceAPIKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	c, cleanup, err := getClientForAPIKeyManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	_, err = c.DeleteAPIKey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getClientForAPIKeyManagement(d *schema.ResourceData, m interface{}) (c *gapi.Client, cleanup func() error, err error) {
	c = m.(*client).gapi
	cleanup = func() error { return nil }
	if cloudStackSlug, ok := d.GetOk("cloud_stack_slug"); ok && cloudStackSlug.(string) != "" {
		cloudClient := m.(*client).gcloudapi
		c, cleanup, err = cloudClient.CreateTemporaryStackGrafanaClient(cloudStackSlug.(string), "terraform-temp-", 60*time.Second)
	}

	return
}
