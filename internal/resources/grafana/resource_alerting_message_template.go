package grafana

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceMessageTemplate() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Alerting notification template groups, including notification templates.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#notification-template-groups)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: common.WithAlertingMutex[schema.CreateContextFunc](putMessageTemplate),
		ReadContext:   readMessageTemplate,
		UpdateContext: common.WithAlertingMutex[schema.UpdateContextFunc](putMessageTemplate),
		DeleteContext: common.WithAlertingMutex[schema.DeleteContextFunc](deleteMessageTemplate),
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
				Description: "The name of the notification template group.",
			},
			"template": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The content of the notification template group.",
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

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_message_template",
		orgResourceIDString("name"),
		schema,
	).WithLister(listerFunctionOrgResource(listMessageTemplate))
}

func listMessageTemplate(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	// Retry if the API returns 500 because it may be that the alertmanager is not ready in the org yet.
	// The alertmanager is provisioned asynchronously when the org is created.
	if err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.GetTemplates()
		if err != nil {
			if orgID > 1 && (err.(*runtime.APIError).IsCode(500) || err.(*runtime.APIError).IsCode(403)) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		for _, template := range resp.Payload {
			ids = append(ids, MakeOrgResourceID(orgID, template.Name))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return ids, nil
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
			params.SetXDisableProvenance(&provenanceDisabled)
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
	client, _, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	params := provisioning.NewDeleteTemplateParams().WithName(name)
	_, err := client.Provisioning.DeleteTemplate(params)
	diag, _ := common.CheckReadError("message template", data, err)
	return diag
}
