package syntheticmonitoring

import (
	"context"
	"fmt"
	"strconv"

	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/synthetic-monitoring-api-go-client/model"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var resourceCheckAlertID = common.NewResourceID(common.IntIDField("check_id"))

const (
	// Alert names
	AlertNameProbeFailedExecutionsTooHigh        = "ProbeFailedExecutionsTooHigh"
	AlertNameTLSTargetCertificateCloseToExpiring = "TLSTargetCertificateCloseToExpiring"
)

func resourceCheckAlerts() *common.Resource {
	return common.NewLegacySDKResource(
		common.CategorySyntheticMonitoring,
		"grafana_synthetic_monitoring_check_alerts",
		resourceCheckAlertID,
		&schema.Resource{
			Description: `
Manages alerts for a check in Grafana Synthetic Monitoring.

* [Official documentation](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/configure-alerts/)
* [API documentation](https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#alerts)
`,

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
								Description: "Name of the alert. Must be one of: " + AlertNameProbeFailedExecutionsTooHigh + ", " + AlertNameTLSTargetCertificateCloseToExpiring,
								Type:        schema.TypeString,
								Required:    true,
								ValidateFunc: validation.StringInSlice([]string{
									AlertNameProbeFailedExecutionsTooHigh,
									AlertNameTLSTargetCertificateCloseToExpiring,
								}, false),
							},
							"threshold": {
								Description: "Threshold value for the alert.",
								Type:        schema.TypeFloat,
								Required:    true,
							},
							"period": {
								Description: "Period for the alert threshold. Required only for " + AlertNameProbeFailedExecutionsTooHigh + " alerts. One of: `1m`, `2m`, `5m`, `10m`, `15m`, `20m`, `30m`, `1h`.",
								Type:        schema.TypeString,
								Required:    false,
								Optional:    true,
								Default:     "",
								ValidateFunc: validation.StringInSlice([]string{
									"", "1m", "2m", "5m", "10m", "15m", "20m", "30m", "1h",
								}, false),
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

	// Validate that the check exists
	if err := validateCheckExists(ctx, c, checkID); err != nil {
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

	d.SetId(fmt.Sprintf("%d", checkID))
	return resourceCheckAlertRead(ctx, d, c)
}

func resourceCheckAlertRead(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	checkID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	// Validate that the check exists
	if err := validateCheckExists(ctx, c, checkID); err != nil {
		return diag.FromErr(err)
	}

	alerts, err := c.GetCheckAlerts(ctx, checkID)
	if err != nil {
		return diag.FromErr(err)
	}

	// Transform alerts into schema format
	alertsList := make([]map[string]interface{}, len(alerts))
	for i, alert := range alerts {
		alertMap := map[string]interface{}{
			"name":      alert.Name,
			"threshold": alert.Threshold,
		}
		if alert.Period != "" {
			alertMap["period"] = alert.Period
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

	// Validate that the check exists
	if err := validateCheckExists(ctx, c, checkID); err != nil {
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

	// Validate that the check exists
	if err := validateCheckExists(ctx, c, checkID); err != nil {
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
		alertData := alertMap.(map[string]interface{})
		name := alertData["name"].(string)
		period, hasPeriod := alertData["period"].(string)

		if name == AlertNameProbeFailedExecutionsTooHigh && !hasPeriod {
			return nil, fmt.Errorf("period is required when name is %s", AlertNameProbeFailedExecutionsTooHigh)
		}

		alert := model.CheckAlert{
			Name:      name,
			Threshold: alertData["threshold"].(float64),
		}

		if hasPeriod {
			alert.Period = period
		}

		alerts[i] = alert
	}

	return alerts, nil
}

func validateCheckExists(ctx context.Context, c *smapi.Client, checkID int64) error {
	_, err := c.GetCheck(ctx, checkID)
	return err
}
