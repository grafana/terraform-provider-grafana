package grafana

import (
	"errors"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceOnCallSchedule() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/oncall/calendar-schedules/)
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/schedules/)
`,
		Read: dataSourceOnCallScheduleRead,
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

func dataSourceOnCallScheduleRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}
	options := &onCallAPI.ListScheduleOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	schedulesResponse, _, err := client.Schedules.ListSchedules(options)
	if err != nil {
		return err
	}

	if len(schedulesResponse.Schedules) == 0 {
		return fmt.Errorf("couldn't find a schedule matching: %s", options.Name)
	} else if len(schedulesResponse.Schedules) != 1 {
		return fmt.Errorf("more than one schedule found matching: %s", options.Name)
	}

	schedule := schedulesResponse.Schedules[0]

	d.SetId(schedule.ID)
	d.Set("type", schedule.Type)

	return nil
}
