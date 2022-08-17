package grafana

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceNotificationPolicy() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/notifications/)
* [HTTP API](https://grafana.com/docs/grafana/next/developers/http_api/alerting_provisioning/#notification-policies)
`,

		CreateContext: createNotificationPolicy,
		ReadContext:   readNotificationPolicy,
		UpdateContext: updateNotificationPolicy,
		DeleteContext: deleteNotificationPolicy,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"contact_point": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The default contact point to route all unmatched notifications to.",
			},
			"group_by": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group alerts by all labels, effectively disabling grouping.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"group_wait": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Time to wait to buffer alerts of the same group before sending a notification. Default is 30 seconds.",
			},
			"group_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Minimum time interval between two notifications for the same group. Default is 5 minutes.",
			},
			"repeat_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Minimum time interval for re-sending a notification if an alert is still firing. Default is 4 hours.",
			},
		},
	}
}

const PolicySingletonID = "policy"

func readNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	npt, err := client.NotificationPolicyTree()
	if err != nil {
		return diag.FromErr(err)
	}

	packNotifPolicy(npt, data)
	data.SetId(PolicySingletonID)
	return nil
}

func createNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	npt := unpackNotifPolicy(data)

	if err := client.SetNotificationPolicyTree(&npt); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(PolicySingletonID)
	return readNotificationPolicy(ctx, data, meta)
}

func updateNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	npt := unpackNotifPolicy(data)

	if err := client.SetNotificationPolicyTree(&npt); err != nil {
		return diag.FromErr(err)
	}

	return readNotificationPolicy(ctx, data, meta)
}

func deleteNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	if err := client.ResetNotificationPolicyTree(); err != nil {
		return diag.FromErr(err)
	}
	return diag.Diagnostics{}
}

func packNotifPolicy(npt gapi.NotificationPolicyTree, data *schema.ResourceData) {
	data.Set("contact_point", npt.Receiver)
	data.Set("group_by", npt.GroupBy)
	data.Set("group_wait", npt.GroupWait)
	data.Set("group_interval", npt.GroupInterval)
	data.Set("repeat_interval", npt.RepeatInterval)
}

func unpackNotifPolicy(data *schema.ResourceData) gapi.NotificationPolicyTree {
	groupBy := data.Get("group_by").([]interface{})
	groups := make([]string, 0, len(groupBy))
	for _, g := range groupBy {
		groups = append(groups, g.(string))
	}
	return gapi.NotificationPolicyTree{
		Receiver:       data.Get("contact_point").(string),
		GroupBy:        groups,
		GroupWait:      data.Get("group_wait").(string),
		GroupInterval:  data.Get("group_interval").(string),
		RepeatInterval: data.Get("repeat_interval").(string),
	}
}
