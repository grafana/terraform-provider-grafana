package cloud

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/api_keys"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceStackAPIKey() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages API keys of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management API key for a new stack

* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/auth/)

Required access policy scopes:

* stack-api-keys:write

!> Deprecated: please use ` + "`grafana_cloud_stack_service_account`" + ` and ` + "`grafana_cloud_stack_service_account_token`" + ` instead, see https://grafana.com/docs/grafana/next/administration/api-keys/#migrate-api-keys-to-grafana-service-accounts-using-terraform.
`,

		CreateContext:      withClient[schema.CreateContextFunc](resourceStackAPIKeyCreate),
		ReadContext:        withClient[schema.ReadContextFunc](resourceStackAPIKeyRead),
		DeleteContext:      withClient[schema.DeleteContextFunc](resourceStackAPIKeyDelete),
		DeprecationMessage: "Use `grafana_cloud_stack_service_account` together with `grafana_cloud_stack_service_account_token` resources instead see https://grafana.com/docs/grafana/next/administration/api-keys/#migrate-api-keys-to-grafana-service-accounts-using-terraform",

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
			"role": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin"}, false),
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
		},
	}

	return common.NewLegacySDKResource(
		"grafana_cloud_stack_api_key",
		nil,
		schema,
	)
}

func resourceStackAPIKeyCreate(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	name := d.Get("name").(string)
	role := d.Get("role").(string)
	ttl := d.Get("seconds_to_live").(int)

	c, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	request := &models.AddAPIKeyCommand{Name: name, Role: role, SecondsToLive: int64(ttl)}
	err = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := c.APIKeys.AddAPIkey(request)

		if err != nil {
			if strings.Contains(err.Error(), "Your instance is loading, and will be ready shortly.") {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}
		key := response.Payload

		d.SetId(strconv.FormatInt(key.ID, 10))
		d.Set("key", key.Key)
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	// Fill the true resource's state after a create by performing a read
	return resourceStackAPIKeyReadWithClient(c, d)
}

func resourceStackAPIKeyRead(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	c, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	return resourceStackAPIKeyReadWithClient(c, d)
}

func resourceStackAPIKeyReadWithClient(c *goapi.GrafanaHTTPAPI, d *schema.ResourceData) diag.Diagnostics {
	response, err := c.APIKeys.GetAPIkeys(api_keys.NewGetAPIkeysParams().WithIncludeExpired(common.Ref((true))))
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

func resourceStackAPIKeyDelete(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	c, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	_, err = c.APIKeys.DeleteAPIkey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
