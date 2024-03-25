package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/client/api_keys"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAPIKey() *common.Resource {
	schema := &schema.Resource{
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

	return common.NewResource(
		"grafana_api_key",
		nil,
		schema,
	)
}

func resourceAPIKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, orgID := OAPIClientFromNewOrgResource(m, d)

	request := models.AddAPIKeyCommand{
		Name:          d.Get("name").(string),
		Role:          d.Get("role").(string),
		SecondsToLive: int64(d.Get("seconds_to_live").(int)),
	}
	response, err := c.APIKeys.AddAPIkey(&request)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, response.Payload.ID))
	d.Set("key", response.Payload.Key)

	// Fill the true resource's state after a create by performing a read
	return resourceAPIKeyRead(ctx, d, m)
}

func resourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, orgID, idStr := OAPIClientFromExistingOrgResource(m, d.Id())

	includeExpired := true
	response, err := c.APIKeys.GetAPIkeys(api_keys.NewGetAPIkeysParams().WithIncludeExpired(&includeExpired))
	if err, shouldReturn := common.CheckReadError("API key", d, err); shouldReturn {
		return err
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, key := range response.Payload {
		if id == key.ID {
			d.SetId(MakeOrgResourceID(orgID, key.ID))
			d.Set("org_id", strconv.FormatInt(orgID, 10))
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
	c, _, idStr := OAPIClientFromExistingOrgResource(m, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.APIKeys.DeleteAPIkey(id)
	diag, _ := common.CheckReadError("API key", d, err)
	return diag
}
