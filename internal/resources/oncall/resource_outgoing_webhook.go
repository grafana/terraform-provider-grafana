package oncall

import (
	"context"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// When a preset is defined, certain fields are controlled by that preset and should not be
// modified by users; these fields are automatically suppressed from diffs when a preset is active.
// presetControlledFields maps preset names to lists of field names that are controlled by that preset
var presetControlledFields = map[string][]string{
	"advanced_webhook": {},
	"grafana_sift":     {"authorization_header", "data", "forward_whole_payload", "headers", "http_method", "password", "url", "user"},
	"incident_webhook": {"integration_filter"},
	"simple_webhook":   {"authorization_header", "data", "forward_whole_payload", "headers", "http_method", "integration_filter", "password", "trigger_template", "trigger_type", "user"},
}

// isFieldControlledByPreset checks if a field is controlled by the current preset
func isFieldControlledByPreset(fieldName string, d *schema.ResourceData) bool {
	if preset, presetOk := d.GetOk("preset"); presetOk {
		if preset == "" {
			// If no preset is set, default to advanced_webhook
			preset = "advanced_webhook"
		}
		if controlledFields, exists := presetControlledFields[preset.(string)]; exists {
			for _, controlledField := range controlledFields {
				if controlledField == fieldName {
					return true
				}
			}
		}
	}
	return false
}

// suppressDiffForPresetControlledField is a generic diff suppression function
// that checks if a field is controlled by the current preset
func suppressDiffForPresetControlledField(fieldName string) func(k, old, new string, d *schema.ResourceData) bool {
	return func(k, old, new string, d *schema.ResourceData) bool {
		return isFieldControlledByPreset(fieldName, d)
	}
}

func resourceOutgoingWebhook() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/outgoing_webhooks/)
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceOutgoingWebhookCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceOutgoingWebhookRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceOutgoingWebhookUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceOutgoingWebhookDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the outgoing webhook.",
			},
			"preset": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The preset of the outgoing webhook.",
			},
			"team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the OnCall team (using the `grafana_oncall_team` datasource).",
			},
			"url": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The webhook URL. Required when not using a preset that controls this field.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("url"),
			},
			"data": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The data of the webhook.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("data"),
			},
			"user": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Username to use when making the outgoing webhook request.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("user"),
			},
			"password": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The auth data of the webhook. Used for Basic authentication",
				Sensitive:        true,
				DiffSuppressFunc: suppressDiffForPresetControlledField("password"),
			},
			"authorization_header": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The auth data of the webhook. Used in Authorization header instead of user/password auth.",
				Sensitive:        true,
				DiffSuppressFunc: suppressDiffForPresetControlledField("authorization_header"),
			},
			"forward_whole_payload": {
				Type:             schema.TypeBool,
				Optional:         true,
				Description:      "Toggle to send the entire webhook payload instead of using the values in the Data field.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("forward_whole_payload"),
			},
			"trigger_type": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The type of event that will cause this outgoing webhook to execute. The types of triggers are: `escalation`, `alert group created`, `acknowledge`, `resolve`, `silence`, `unsilence`, `unresolve`, `unacknowledge`.",
				Default:          "escalation",
				DiffSuppressFunc: suppressDiffForPresetControlledField("trigger_type"),
			},
			"http_method": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The HTTP method used in the request made by the outgoing webhook.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("http_method"),
			},
			"trigger_template": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "A template used to dynamically determine whether the webhook should execute based on the content of the payload.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("trigger_template"),
			},
			"headers": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Headers to add to the outgoing webhook request.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("headers"),
			},
			"integration_filter": {
				Type:             schema.TypeList,
				Elem:             &schema.Schema{Type: schema.TypeString},
				Optional:         true,
				Description:      "Restricts the outgoing webhook to only trigger if the event came from a selected integration. If no integrations are selected the outgoing webhook will trigger for any integration.",
				DiffSuppressFunc: suppressDiffForPresetControlledField("integration_filter"),
			},
			"is_webhook_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Controls whether the outgoing webhook will trigger or is ignored.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryOnCall,
		"grafana_oncall_outgoing_webhook",
		resourceID,
		schema,
	).
		WithLister(oncallListerFunction(listWebhooks)).
		WithPreferredResourceNameField("name")
}

func listWebhooks(client *onCallAPI.Client, listOptions onCallAPI.ListOptions) (ids []string, nextPage *string, err error) {
	resp, _, err := client.Webhooks.ListWebhooks(&onCallAPI.ListWebhookOptions{ListOptions: listOptions})
	if err != nil {
		return nil, nil, err
	}
	for _, i := range resp.Webhooks {
		ids = append(ids, i.ID)
	}
	return ids, resp.Next, nil
}

func resourceOutgoingWebhookCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	name := d.Get("name").(string)
	teamID := d.Get("team_id").(string)
	forwardWholePayload := d.Get("forward_whole_payload").(bool)
	isWebhookEnabled := d.Get("is_webhook_enabled").(bool)

	createOptions := &onCallAPI.CreateWebhookOptions{
		Name:             name,
		Team:             teamID,
		ForwardAll:       forwardWholePayload,
		IsWebhookEnabled: isWebhookEnabled,
	}

	preset, presetOk := d.GetOk("preset")
	if presetOk {
		createOptions.Preset = preset.(string)
	}

	// Handle URL validation and assignment
	if !isFieldControlledByPreset("url", d) {
		url, urlOk := d.GetOk("url")
		if !urlOk || url == "" {
			return diag.Errorf("url is required if it is not defined by the preset")
		}
		createOptions.Url = url.(string)
	}

	data, dataOk := d.GetOk("data")
	if dataOk {
		dd := data.(string)
		createOptions.Data = &dd
	}
	user, userOk := d.GetOk("user")
	if userOk {
		u := user.(string)
		createOptions.Username = &u
	}

	password, passwordOk := d.GetOk("password")
	if passwordOk {
		p := password.(string)
		createOptions.Password = &p
	}
	authHeader, authHeaderOk := d.GetOk("authorization_header")
	if authHeaderOk {
		a := authHeader.(string)
		createOptions.AuthorizationHeader = &a
	}

	triggerType, triggerTypeOk := d.GetOk("trigger_type")
	if triggerTypeOk {
		createOptions.TriggerType = triggerType.(string)
	}

	httpMethod, httpMethodOk := d.GetOk("http_method")
	if httpMethodOk {
		createOptions.HttpMethod = httpMethod.(string)
	}

	triggerTemplate, triggerTemplateOk := d.GetOk("trigger_template")
	if triggerTemplateOk {
		t := triggerTemplate.(string)
		createOptions.TriggerTemplate = &t
	}

	headers, headersOk := d.GetOk("headers")
	if headersOk {
		h := headers.(string)
		createOptions.Headers = &h
	}

	integrationFilter, integrationFilterOk := d.GetOk("integration_filter")
	if integrationFilterOk {
		f := integrationFilter.([]interface{})
		integrationFilterSlice := make([]string, len(f))
		for i := range f {
			integrationFilterSlice[i] = f[i].(string)
		}
		createOptions.IntegrationFilter = &integrationFilterSlice
	}

	outgoingWebhook, _, err := client.Webhooks.CreateWebhook(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(outgoingWebhook.ID)

	return resourceOutgoingWebhookRead(ctx, d, client)
}

func resourceOutgoingWebhookRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	outgoingWebhook, r, err := client.Webhooks.GetWebhook(d.Id(), &onCallAPI.GetWebhookOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			return common.WarnMissing("outgoing webhook", d)
		}
		return diag.FromErr(err)
	}

	d.Set("name", outgoingWebhook.Name)
	d.Set("preset", outgoingWebhook.Preset)
	d.Set("team_id", outgoingWebhook.Team)
	d.Set("url", outgoingWebhook.Url)
	d.Set("data", outgoingWebhook.Data)
	d.Set("user", outgoingWebhook.Username)
	d.Set("forward_whole_payload", outgoingWebhook.ForwardAll)
	d.Set("is_webhook_enabled", outgoingWebhook.IsWebhookEnabled)
	d.Set("trigger_type", outgoingWebhook.TriggerType)
	d.Set("http_method", outgoingWebhook.HttpMethod)
	d.Set("trigger_template", outgoingWebhook.TriggerTemplate)
	d.Set("headers", outgoingWebhook.Headers)
	d.Set("integration_filter", outgoingWebhook.IntegrationFilter)

	return nil
}

func resourceOutgoingWebhookUpdate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	name := d.Get("name").(string)
	teamID := d.Get("team_id").(string)
	forwardWholePayload := d.Get("forward_whole_payload").(bool)
	isWebhookEnabled := d.Get("is_webhook_enabled").(bool)

	updateOptions := &onCallAPI.UpdateWebhookOptions{
		Name:             name,
		Team:             teamID,
		ForwardAll:       forwardWholePayload,
		IsWebhookEnabled: isWebhookEnabled,
	}

	preset, presetOk := d.GetOk("preset")
	if presetOk {
		updateOptions.Preset = preset.(string)
	}

	// Handle URL validation and assignment
	if !isFieldControlledByPreset("url", d) {
		url, urlOk := d.GetOk("url")
		if !urlOk || url == "" {
			return diag.Errorf("url is required if it is not defined by the preset")
		}
		updateOptions.Url = url.(string)
	}

	data, dataOk := d.GetOk("data")
	if dataOk {
		dd := data.(string)
		updateOptions.Data = &dd
	}
	user, userOk := d.GetOk("user")
	if userOk {
		u := user.(string)
		updateOptions.Username = &u
	}

	password, passwordOk := d.GetOk("password")
	if passwordOk {
		p := password.(string)
		updateOptions.Password = &p
	}
	authHeader, authHeaderOk := d.GetOk("authorization_header")
	if authHeaderOk {
		a := authHeader.(string)
		updateOptions.AuthorizationHeader = &a
	}

	triggerType, triggerTypeOk := d.GetOk("trigger_type")
	if triggerTypeOk {
		updateOptions.TriggerType = triggerType.(string)
	}

	httpMethod, httpMethodOk := d.GetOk("http_method")
	if httpMethodOk {
		updateOptions.HttpMethod = httpMethod.(string)
	}

	triggerTemplate, triggerTemplateOk := d.GetOk("trigger_template")
	if triggerTemplateOk {
		t := triggerTemplate.(string)
		updateOptions.TriggerTemplate = &t
	}

	headers, headersOk := d.GetOk("headers")
	if headersOk {
		h := headers.(string)
		updateOptions.Headers = &h
	}

	integrationFilter, integrationFilterOk := d.GetOk("integration_filter")
	if integrationFilterOk {
		f := integrationFilter.([]interface{})
		integrationFilterSlice := make([]string, len(f))
		for i := range f {
			integrationFilterSlice[i] = f[i].(string)
		}
		updateOptions.IntegrationFilter = &integrationFilterSlice
	}

	outgoingWebhook, _, err := client.Webhooks.UpdateWebhook(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(outgoingWebhook.ID)
	return resourceOutgoingWebhookRead(ctx, d, client)
}

func resourceOutgoingWebhookDelete(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	_, err := client.Webhooks.DeleteWebhook(d.Id(), &onCallAPI.DeleteWebhookOptions{})
	return diag.FromErr(err)
}
