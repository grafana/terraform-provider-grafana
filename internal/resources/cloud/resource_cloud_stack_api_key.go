package cloud

import (
	"context"
	"strconv"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceStackAPIKey() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages API keys of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management API key for a new stack

* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/auth/)
`,

		CreateContext: resourceStackAPIKeyCreate,
		ReadContext:   resourceStackAPIKeyRead,
		DeleteContext: resourceStackAPIKeyDelete,

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
}

func resourceStackAPIKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	role := d.Get("role").(string)
	ttl := d.Get("seconds_to_live").(int)

	c, cleanup, err := getClientForAPIKeyManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	request := gapi.CreateAPIKeyRequest{Name: name, Role: role, SecondsToLive: int64(ttl)}
	err = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := c.CreateAPIKey(request)

		if err != nil {
			if strings.Contains(err.Error(), "Your instance is loading, and will be ready shortly.") {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		d.SetId(strconv.FormatInt(response.ID, 10))
		d.Set("key", response.Key)
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	// Fill the true resource's state after a create by performing a read
	return resourceStackAPIKeyRead(ctx, d, m)
}

func resourceStackAPIKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

func resourceStackAPIKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
	cloudClient := m.(*common.Client).GrafanaCloudAPI
	return cloudClient.CreateTemporaryStackGrafanaClient(d.Get("stack_slug").(string), "terraform-temp-", 60*time.Second)
}
