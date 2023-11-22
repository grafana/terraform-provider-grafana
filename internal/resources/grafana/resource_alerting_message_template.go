package grafana

import (
	"context"
	"strings"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
		CreateContext: createMessageTemplate,
		ReadContext:   readMessageTemplate,
		UpdateContext: updateMessageTemplate,
		DeleteContext: deleteMessageTemplate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
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
		},
	}
}

func readMessageTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	name := data.Id()
	tmpl, err := client.MessageTemplate(name)
	if err, shouldReturn := common.CheckReadError("message template", data, err); shouldReturn {
		return err
	}

	data.SetId(tmpl.Name)
	data.Set("name", tmpl.Name)
	data.Set("template", tmpl.Template)

	return nil
}

func createMessageTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI
	name := data.Get("name").(string)
	content := data.Get("template").(string)

	lock.Lock()
	defer lock.Unlock()
	if err := client.SetMessageTemplate(name, content); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(name)
	return readMessageTemplate(ctx, data, meta)
}

func updateMessageTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI
	name := data.Get("name").(string)
	content := data.Get("template").(string)

	lock.Lock()
	defer lock.Unlock()
	if err := client.SetMessageTemplate(name, content); err != nil {
		return diag.FromErr(err)
	}

	return readMessageTemplate(ctx, data, meta)
}

func deleteMessageTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI
	name := data.Id()

	lock.Lock()
	defer lock.Unlock()
	err := client.DeleteMessageTemplate(name)
	diag, _ := common.CheckReadError("message template", data, err)
	return diag
}
