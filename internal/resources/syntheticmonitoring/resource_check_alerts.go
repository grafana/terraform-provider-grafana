package syntheticmonitoring

import (
	"context"
	"strconv"
	"strings"

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
								Description: "URL to runbook documentation for this alert.",
								Type:        schema.TypeString,
								Optional:    true,
								Default:     "",
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
		// Alerts have no existence independent of their check. If the check is gone,
		// the alerts are gone with it, so treat this like any other missing resource
		// rather than failing the refresh.
		if isCheckNotFound(err) {
			return common.WarnMissing("check alerts", d)
		}
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
		alertMap["runbook_url"] = alert.RunbookUrl
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
		// If the check has already been deleted, its alerts were removed with it and
		// there is nothing left to do. Without this, deleting a check before its
		// alerts leaves the alerts resource permanently undeletable: every retry
		// hits the same 404. Crossplane's provider-grafana wraps this resource and
		// surfaces exactly that as a finalizer that never releases.
		if isCheckNotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

// isCheckNotFound reports whether the SM API rejected a check-alerts operation
// because the parent check no longer exists. The client returns the HTTP status
// and the API message in the error text; this matches the same way the check and
// probe resources in this package detect a missing resource.
func isCheckNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "404 Not Found") || strings.Contains(msg, "check not found")
}

func makeCheckAlerts(d *schema.ResourceData) ([]model.CheckAlert, error) {
	alertsSet := d.Get("alerts").(*schema.Set)
	alertsList := alertsSet.List()
	alerts := make([]model.CheckAlert, len(alertsList))

	for i, alertMap := range alertsList {
		alertData := alertMap.(map[string]any)
		name := alertData["name"].(string)
		period, hasPeriod := alertData["period"].(string)
		runbookURL, hasRunbookURL := alertData["runbook_url"].(string)

		alert := model.CheckAlert{
			Name:      name,
			Threshold: alertData["threshold"].(float64),
		}

		if hasPeriod {
			alert.Period = period
		}

		if hasRunbookURL {
			alert.RunbookUrl = runbookURL
		}

		alerts[i] = alert
	}

	return alerts, nil
}
