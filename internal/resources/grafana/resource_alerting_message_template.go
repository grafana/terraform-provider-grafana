package grafana

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceMessageTemplate() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana Alerting message templates.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/manage-notifications/template-notifications/create-notification-templates/)
* [HTTP API](https://grafana.com/docs/grafana/next/developers/http_api/alerting_provisioning/#templates)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: putMessageTemplate,
		ReadContext:   readMessageTemplate,
		UpdateContext: putMessageTemplate,
		DeleteContext: deleteMessageTemplate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the message template.",
			},
			"template": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The content of the message template.",
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
			},
			"disable_provenance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true, // TODO: The API doesn't return provenance, so we have to force new for now.
				Description: "Allow modifying the message template from other sources than Terraform or the Grafana API.",
			},
		},
	}
}

func readMessageTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	resp, err := client.Provisioning.GetTemplate(name)
	if err, shouldReturn := common.CheckReadError("message template", data, err); shouldReturn {
		return err
	}
	tmpl := resp.Payload

	data.Set("org_id", strconv.FormatInt(orgID, 10))
	data.Set("name", tmpl.Name)
	data.Set("template", tmpl.Template)

	return nil
}

func putMessageTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	lock.Lock()
	defer lock.Unlock()
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	name := data.Get("name").(string)
	content := data.Get("template").(string)

	// Retry if the API returns 500 because it may be that the alertmanager is not ready in the org yet.
	// The alertmanager is provisioned asynchronously when the org is created.
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		params := provisioning.NewPutTemplateParams().
			WithName(name).
			WithBody(&models.NotificationTemplateContent{
				Template: content,
			})
		if v, ok := data.GetOk("disable_provenance"); ok && v.(bool) {
			disabled := "disabled"
			params.SetXDisableProvenance(&disabled)
		}
		if _, err := client.Provisioning.PutTemplate(params); err != nil {
			if orgID > 1 && err.(*runtime.APIError).IsCode(500) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(MakeOrgResourceID(orgID, name))
	return readMessageTemplate(ctx, data, meta)
}

func deleteMessageTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	lock.Lock()
	defer lock.Unlock()
	client, _, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	_, err := client.Provisioning.DeleteTemplate(name)
	diag, _ := common.CheckReadError("message template", data, err)
	return diag
}
