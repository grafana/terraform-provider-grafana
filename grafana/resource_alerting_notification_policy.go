package grafana

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceNotificationPolicy() *schema.Resource {
	return &schema.Resource{
		Description: `TODO`,

		ReadContext: readNotificationPolicy,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"receiver": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The default contact point to fall back to.",
			},
			"group_by": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group alerts by all labels, effectively disabling grouping.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func readNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	npt, err := client.NotificationPolicyTree()
	if err != nil {
		return diag.FromErr(err)
	}

	packNotifPolicy(npt, data)
	data.SetId("TODO")
	return nil
}

func packNotifPolicy(npt gapi.NotificationPolicyTree, data *schema.ResourceData) {
	data.Set("receiver", npt.Receiver)
	data.Set("group_by", npt.GroupBy)
}
