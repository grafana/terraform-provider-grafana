package cloud

import (
	"context"
	"log"
	"strconv"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceStackServiceAccountToken() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages service account tokens of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management service account token for a new stack

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)

Required access policy scopes:

* stack-service-accounts:write
`,

		CreateContext: withClient[schema.CreateContextFunc](stackServiceAccountTokenCreate),
		ReadContext:   withClient[schema.ReadContextFunc](stackServiceAccountTokenRead),
		DeleteContext: withClient[schema.DeleteContextFunc](stackServiceAccountTokenDelete),

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

	return common.NewResource(
		"grafana_cloud_stack_service_account_token",
		nil,
		schema,
	)
}

func stackServiceAccountTokenCreate(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	c, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	split, err := resourceStackServiceAccountID.Split(d.Get("service_account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := split[1].(int64)

	name := d.Get("name").(string)
	ttl := d.Get("seconds_to_live").(int)

	request := service_accounts.NewCreateTokenParams().WithBody(&models.AddServiceAccountTokenCommand{
		Name:          name,
		SecondsToLive: int64(ttl),
	}).WithServiceAccountID(serviceAccountID)
	response, err := c.ServiceAccounts.CreateToken(request)
	if err != nil {
		return diag.FromErr(err)
	}
	t := response.Payload

	d.SetId(strconv.FormatInt(t.ID, 10))
	err = d.Set("key", t.Key)
	if err != nil {
		return diag.FromErr(err)
	}

	// Fill the true resource's state by performing a read
	return stackServiceAccountTokenReadWithClient(c, d)
}

func stackServiceAccountTokenRead(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	c, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	return stackServiceAccountTokenReadWithClient(c, d)
}

func stackServiceAccountTokenReadWithClient(c *goapi.GrafanaHTTPAPI, d *schema.ResourceData) diag.Diagnostics {
	split, err := resourceStackServiceAccountID.Split(d.Get("service_account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := split[1].(int64)

	response, err := c.ServiceAccounts.ListTokens(serviceAccountID)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, key := range response.Payload {
		if id == key.ID {
			d.SetId(strconv.FormatInt(key.ID, 10))
			err = d.Set("name", key.Name)
			if err != nil {
				return diag.FromErr(err)
			}
			if !key.Expiration.IsZero() {
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

func stackServiceAccountTokenDelete(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	c, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	split, err := resourceStackServiceAccountID.Split(d.Get("service_account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := split[1].(int64)

	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.ServiceAccounts.DeleteToken(id, serviceAccountID)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
