package cloud

import (
	"context"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"strconv"
	"time"
)

func ResourceStackServiceAccountToken() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana 9.1+.

Manages service account tokens of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management service account token for a new stack

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)`,

		CreateContext: stackServiceAccountTokenCreate,
		ReadContext:   stackServiceAccountTokenRead,
		DeleteContext: stackServiceAccountTokenDelete,

		Schema: map[string]*schema.Schema{
			"stack_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account_id": {
				Type:     schema.TypeString,
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

func stackServiceAccountTokenCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, cleanup, err := getClientForSATokenManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	serviceAccountID, err := strconv.ParseInt(d.Get("service_account_id").(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	ttl := d.Get("seconds_to_live").(int)

	request := gapi.CreateServiceAccountTokenRequest{
		Name:             name,
		ServiceAccountID: serviceAccountID,
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
	return stackServiceAccountTokenRead(ctx, d, m)
}

func stackServiceAccountTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, cleanup, err := getClientForSATokenManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	serviceAccountID, err := strconv.ParseInt(d.Get("service_account_id").(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

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

	log.Printf("[WARN] removing service account token%d from state because it no longer exists in grafana", id)
	d.SetId("")

	return nil
}

func stackServiceAccountTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, cleanup, err := getClientForSATokenManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	serviceAccountID, err := strconv.ParseInt(d.Get("service_account_id").(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.DeleteServiceAccountToken(serviceAccountID, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getClientForSATokenManagement(d *schema.ResourceData, m interface{}) (c *gapi.Client, cleanup func() error, err error) {
	cloudClient := m.(*common.Client).GrafanaCloudAPI
	return cloudClient.CreateTemporaryStackGrafanaClient(d.Get("stack_slug").(string), "terraform-temp-sa-token-", 60*time.Second)
}
