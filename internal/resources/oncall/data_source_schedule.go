package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceSchedule() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/oncall/calendar-schedules/)
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/schedules/)
`,
		ReadContext: DataSourceScheduleRead,
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
}

func DataSourceScheduleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient
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
