package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSchedule() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/oncall/latest/manage/on-call-schedules/)
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/schedules/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceScheduleRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The schedule name.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The schedule type.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_schedule", schema)
}

func dataSourceScheduleRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.ListScheduleOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	schedulesResponse, _, err := client.Schedules.ListSchedules(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(schedulesResponse.Schedules) == 0 {
		return diag.Errorf("couldn't find a schedule matching: %s", options.Name)
	} else if len(schedulesResponse.Schedules) != 1 {
		return diag.Errorf("more than one schedule found matching: %s", options.Name)
	}

	schedule := schedulesResponse.Schedules[0]

	d.SetId(schedule.ID)
	d.Set("type", schedule.Type)

	return nil
}
