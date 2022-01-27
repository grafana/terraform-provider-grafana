package grafana

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	ErrFrequencyMustBeSet = errors.New("frequency must be set when send_reminder is set to 'true'")
)

func ResourceAlertNotification() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/notifications/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/alerting_notification_channels/)
`,

		CreateContext: CreateAlertNotification,
		UpdateContext: UpdateAlertNotification,
		DeleteContext: DeleteAlertNotification,
		ReadContext:   ReadAlertNotification,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The type of the alert notification channel.",
			},

			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the alert notification channel.",
			},

			"is_default": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Is this the default channel for all your alerts.",
			},

			"send_reminder": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to send reminders for triggered alerts.",
			},

			"frequency": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Frequency of alert reminders. Frequency must be set if reminders are enabled.",
			},

			"settings": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Additional settings, for full reference see [Grafana HTTP API documentation](https://grafana.com/docs/grafana/latest/http_api/alerting_notification_channels/).",
			},

			"secure_settings": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Description: "Additional secure settings, for full reference lookup [Grafana Supported Settings documentation](https://grafana.com/docs/grafana/latest/administration/provisioning/#supported-settings).",
			},

			"uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Unique identifier. If unset, this will be automatically generated.",
			},

			"disable_resolve_message": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to disable sending resolve messages.",
			},
		},
	}
}

func CreateAlertNotification(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	alertNotification, err := makeAlertNotification(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := client.NewAlertNotification(alertNotification)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(id, 10))

	return ReadAlertNotification(ctx, d, meta)
}

func UpdateAlertNotification(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	alertNotification, err := makeAlertNotification(ctx, d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = client.UpdateAlertNotification(alertNotification); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ReadAlertNotification(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("Invalid id: %#v", idStr)
	}

	alertNotification, err := client.AlertNotification(id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing datasource %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	settings := map[string]interface{}{}
	for k, v := range alertNotification.Settings.(map[string]interface{}) {
		boolVal, ok := v.(bool)
		switch {
		case ok && boolVal:
			settings[k] = "true"
		case ok && !boolVal:
			settings[k] = "false"
		default:
			settings[k] = v
		}
	}
	secureSettings := map[string]interface{}{}

	for k, v := range alertNotification.SecureFields.(map[string]interface{}) {
		boolVal, ok := v.(bool)
		switch {
		case ok && boolVal:
			secureSettings[k] = "true"
		case ok && !boolVal:
			secureSettings[k] = "false"
		default:
			secureSettings[k] = v
		}
	}
	d.Set("secure_settings", secureSettings)
	d.SetId(strconv.FormatInt(alertNotification.ID, 10))
	d.Set("is_default", alertNotification.IsDefault)
	d.Set("name", alertNotification.Name)
	d.Set("type", alertNotification.Type)
	d.Set("settings", settings)
	d.Set("uid", alertNotification.UID)
	d.Set("disable_resolve_message", alertNotification.DisableResolveMessage)
	d.Set("send_reminder", alertNotification.SendReminder)
	d.Set("frequency", alertNotification.Frequency)

	return nil
}

func DeleteAlertNotification(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.Errorf("Invalid id: %#v", idStr)
	}

	if err = client.DeleteAlertNotification(id); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func makeAlertNotification(_ context.Context, d *schema.ResourceData) (*gapi.AlertNotification, error) {
	idStr := d.Id()
	var id int64
	var err error
	if idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	settings := map[string]interface{}{}
	for k, v := range d.Get("settings").(map[string]interface{}) {
		strVal, ok := v.(string)
		switch {
		case ok && strVal == "true":
			settings[k] = true
		case ok && strVal == "false":
			settings[k] = false
		default:
			settings[k] = v
		}
	}
	secureSettings := map[string]interface{}{}
	for k, v := range d.Get("secure_settings").(map[string]interface{}) {
		strVal, ok := v.(string)
		if !ok {
			return nil, errors.New("secure_settings must be a map of string")
		}
		secureSettings[k] = strVal
	}

	sendReminder := d.Get("send_reminder").(bool)
	frequency := d.Get("frequency").(string)

	if sendReminder {
		if frequency == "" {
			return nil, ErrFrequencyMustBeSet
		}

		if _, err := time.ParseDuration(frequency); err != nil {
			return nil, err
		}
	}

	return &gapi.AlertNotification{
		ID:                    id,
		Name:                  d.Get("name").(string),
		Type:                  d.Get("type").(string),
		IsDefault:             d.Get("is_default").(bool),
		DisableResolveMessage: d.Get("disable_resolve_message").(bool),
		UID:                   d.Get("uid").(string),
		SendReminder:          sendReminder,
		Frequency:             frequency,
		Settings:              settings,
		SecureSettings:        secureSettings,
	}, err
}
