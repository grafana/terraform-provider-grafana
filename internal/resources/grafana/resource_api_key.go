package grafana

import (
	"context"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceAPIKey() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana API Keys.

* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/auth/)

!> Deprecated: please use ` + "`grafana_service_account`" + ` and ` + "`grafana_service_account_token`" + ` instead, see [Migrate API keys to Grafana service accounts using Terraform](https://grafana.com/docs/grafana/latest/administration/api-keys/#migrate-api-keys-to-grafana-service-accounts-using-terraform) for more information.
`,

		CreateContext:      resourceAPIKeyCreate,
		ReadContext:        resourceAPIKeyRead,
		DeleteContext:      resourceAPIKeyDelete,
		DeprecationMessage: "Use `grafana_service_account` together with `grafana_service_account_token` instead, see https://grafana.com/docs/grafana/next/administration/api-keys/#migrate-api-keys-to-grafana-service-accounts-using-terraform",

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
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

func resourceAPIKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, orgID := ClientFromNewOrgResource(m, d)

	request := gapi.CreateAPIKeyRequest{
		Name:          d.Get("name").(string),
		Role:          d.Get("role").(string),
		SecondsToLive: int64(d.Get("seconds_to_live").(int)),
	}
	response, err := c.CreateAPIKey(request)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, response.ID))
	d.Set("key", response.Key)

	// Fill the true resource's state after a create by performing a read
	return resourceAPIKeyRead(ctx, d, m)
}

func resourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, _, idStr := ClientFromExistingOrgResource(m, d.Id())

	response, err := c.GetAPIKeys(true)
	if err, shouldReturn := common.CheckReadError("API key", d, err); shouldReturn {
		return err
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, key := range response {
		if id == key.ID {
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
	c, _, idStr := ClientFromExistingOrgResource(m, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.DeleteAPIKey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
