package syntheticmonitoring

import (
	"context"
	"strconv"

	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/synthetic-monitoring-api-go-client/model"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var resourceCheckAlertID = common.NewResourceID(common.IntIDField("check_id"))

const (
	// Alert names
	AlertNameProbeFailedExecutionsTooHigh        = "ProbeFailedExecutionsTooHigh"
	AlertNameTLSTargetCertificateCloseToExpiring = "TLSTargetCertificateCloseToExpiring"
	AlertNameHTTPRequestDurationTooHighAvg       = "HTTPRequestDurationTooHighAvg"
	AlertNamePingRequestDurationTooHighAvg       = "PingRequestDurationTooHighAvg"
	AlertNameDNSRequestDurationTooHighAvg        = "DNSRequestDurationTooHighAvg"
)

func resourceCheckAlerts() *common.Resource {
	return common.NewLegacySDKResource(
		common.CategorySyntheticMonitoring,
		"grafana_synthetic_monitoring_check_alerts",
		resourceCheckAlertID,
		&schema.Resource{
			Description: `
Manages alerts for a check in Grafana Synthetic Monitoring.

* [Official documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/configure-alerts/configure-per-check-alerts/)`,

			CreateContext: withClient[schema.CreateContextFunc](resourceCheckAlertCreate),
			ReadContext:   withClient[schema.ReadContextFunc](resourceCheckAlertRead),
			UpdateContext: withClient[schema.UpdateContextFunc](resourceCheckAlertUpdate),
			DeleteContext: withClient[schema.DeleteContextFunc](resourceCheckAlertDelete),

			Importer: &schema.ResourceImporter{
				StateContext: schema.ImportStatePassthroughContext,
			},

			Schema: map[string]*schema.Schema{
				"check_id": {
					Description: "The ID of the check to manage alerts for.",
					Type:        schema.TypeInt,
					Required:    true,
					ForceNew:    true,
				},
				"alerts": {
					Description: "List of alerts for the check.",
					Type:        schema.TypeSet,
					Required:    true,
					ConfigMode:  schema.SchemaConfigModeAttr,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Description: "Name of the alert. Required.",
								Type:        schema.TypeString,
								Required:    true,
								ValidateFunc: validation.StringInSlice([]string{
									AlertNameProbeFailedExecutionsTooHigh,
									AlertNameTLSTargetCertificateCloseToExpiring,
									AlertNameHTTPRequestDurationTooHighAvg,
									AlertNamePingRequestDurationTooHighAvg,
									AlertNameDNSRequestDurationTooHighAvg,
								}, false),
							},
							"threshold": {
								Description: "Threshold value for the alert.",
								Type:        schema.TypeFloat,
								Required:    true,
							},
							"period": {
								Description: "Period for the alert. Required and must be one of: `5m`, `10m`, `15m`, `20m`, `30m`, `1h`.",
								Type:        schema.TypeString,
								Required:    false,
								Optional:    true,
								Default:     "",
								ValidateFunc: validation.StringInSlice([]string{
									"", "5m", "10m", "15m", "20m", "30m", "1h",
								}, false),
							},
							"runbook_url": {
								Description:  "URL to runbook documentation for this alert.",
								Type:         schema.TypeString,
								Optional:     true,
								Default:      "",
								ValidateFunc: validation.IsURLWithHTTPorHTTPS,
							},
						},
					},
				},
			},
		},
	)
}

func resourceCheckAlertCreate(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	checkID := int64(d.Get("check_id").(int))

	alerts, err := makeCheckAlerts(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.UpdateCheckAlerts(ctx, checkID, alerts)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(checkID, 10))
	return resourceCheckAlertRead(ctx, d, c)
}

func resourceCheckAlertRead(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	checkID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("check_id", checkID)

	alerts, err := c.GetCheckAlerts(ctx, checkID)
	if err != nil {
		return diag.FromErr(err)
	}

	// Transform alerts into schema format
	alertsList := make([]map[string]any, len(alerts))
	for i, alert := range alerts {
		alertMap := map[string]any{
			"name":      alert.Name,
			"threshold": alert.Threshold,
		}
		if alert.Period != "" {
			alertMap["period"] = alert.Period
		}
		if alert.RunbookUrl != "" {
			alertMap["runbook_url"] = alert.RunbookUrl
		}
		alertsList[i] = alertMap
	}

	if err := d.Set("alerts", alertsList); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCheckAlertUpdate(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	checkID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	alerts, err := makeCheckAlerts(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.UpdateCheckAlerts(ctx, checkID, alerts)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCheckAlertRead(ctx, d, c)
}

func resourceCheckAlertDelete(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	checkID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	// Delete all alerts by setting an empty list
	_, err = c.UpdateCheckAlerts(ctx, checkID, []model.CheckAlert{})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func makeCheckAlerts(d *schema.ResourceData) ([]model.CheckAlert, error) {
	alertsSet := d.Get("alerts").(*schema.Set)
	alertsList := alertsSet.List()
	alerts := make([]model.CheckAlert, len(alertsList))

	for i, alertMap := range alertsList {
		alertData := alertMap.(map[string]any)
		name := alertData["name"].(string)
		period, hasPeriod := alertData["period"].(string)
		runbookUrl, hasRunbookUrl := alertData["runbook_url"].(string)

		alert := model.CheckAlert{
			Name:      name,
			Threshold: alertData["threshold"].(float64),
		}

		if hasPeriod {
			alert.Period = period
		}

		if hasRunbookUrl && runbookUrl != "" {
			alert.RunbookUrl = runbookUrl
		}

		alerts[i] = alert
	}

	return alerts, nil
}
