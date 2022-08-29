package grafana

import (
	"context"
	"fmt"
	"log"

	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSyntheticMonitoringInstallation() *schema.Resource {
	return &schema.Resource{

		Description: `
Sets up Synthetic Monitoring on a Grafana cloud stack and generates a token. 
Once a Grafana Cloud stack is created, a user can either use this resource or go into the UI to install synthetic monitoring.
This resource cannot be imported but it can be used on an existing Synthetic Monitoring installation without issues.

* [Official documentation](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/installation/)
* [API documentation](https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#apiv1registerinstall)
`,
		CreateContext: ResourceSyntheticMonitoringInstallationCreate,
		DeleteContext: ResourceSyntheticMonitoringInstallationDelete,

		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics { return nil },
		Schema: map[string]*schema.Schema{
			"metrics_publisher_key": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				ForceNew:    true,
				Description: "The Cloud API Key with the `MetricsPublisher` role used to publish metrics to the SM API",
			},
			"stack_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the stack to install SM on.",
			},
			"metrics_instance_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the metrics instance to install SM on (stack's `prometheus_user_id` attribute).",
			},
			"logs_instance_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the logs instance to install SM on (stack's `logs_user_id` attribute).",
			},
			"sm_access_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Generated token to access the SM API.",
			},
		},
	}
}

func ResourceSyntheticMonitoringInstallationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := smapi.NewClient(meta.(*client).smURL, "", nil)
	stackID, metricsID, logsID := d.Get("stack_id").(int), d.Get("metrics_instance_id").(int), d.Get("logs_instance_id").(int)
	resp, err := c.Install(ctx, int64(stackID), int64(metricsID), int64(logsID), d.Get("metrics_publisher_key").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%d-%d-%d", stackID, metricsID, logsID))
	d.Set("sm_access_token", resp.AccessToken)
	return nil
}

// Management of the installation is a one-off operation. The state cannot be updated through a read operation.
// This read function will only invalidate the state (forcing recreation) if the installation has been deleted.
func ResourceSyntheticMonitoringInstallationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*client)
	tempClient := smapi.NewClient(provider.smURL, d.Get("sm_access_token").(string), nil)
	if err := tempClient.ValidateToken(ctx); err != nil {
		log.Printf("[WARN] removing SM installation from state because it is no longer valid")
		d.SetId("")
	}

	return nil
}

func ResourceSyntheticMonitoringInstallationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*client)
	tempClient := smapi.NewClient(provider.smURL, d.Get("sm_access_token").(string), nil)
	if err := tempClient.DeleteToken(ctx); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
