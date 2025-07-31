package machinelearning

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var resourceHolidayID = common.NewResourceID(common.StringIDField("id"))

func resourceHoliday() *common.Resource {
	schema := &schema.Resource{

		Description: `
A holiday describes time periods where a time series is expected to behave differently to normal.

To use a holiday in a job, use its id in the ` + "`holidays`" + ` attribute of a ` + "`grafana_machine_learning_job`" + `:

` + "```terraform" + `
resource "grafana_machine_learning_job" "test_job" {
  ...
  holidays = [
    grafana_machine_learning_holiday.my_holiday.id
  ]
}
` + "```",

		CreateContext: checkClient(resourceHolidayCreate),
		ReadContext:   checkClient(resourceHolidayRead),
		UpdateContext: checkClient(resourceHolidayUpdate),
		DeleteContext: checkClient(resourceHolidayDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of the holiday.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "The name of the holiday.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A description of the holiday.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"ical_url": {
				Description:  "A URL to an iCal file containing all occurrences of the holiday.",
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"ical_timezone"},
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				AtLeastOneOf: []string{"custom_periods", "ical_url"},
			},
			"ical_timezone": {
				Description:  "The timezone to use for events in the iCal file pointed to by ical_url.",
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"ical_url"},
				ValidateFunc: func(i interface{}, k string) (_ []string, errors []error) {
					_, err := time.LoadLocation(i.(string))
					if err != nil {
						errors = append(errors, fmt.Errorf("expected %q to be a valid IANA Time Zone, got %v: %+v", k, i, err))
					}
					return
				},
			},
			"custom_periods": {
				Description: "A list of custom periods for the holiday.",
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The name of the custom period.",
						},
						"start_time": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsRFC3339Time,
						},
						"end_time": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsRFC3339Time,
						},
					},
				},
				Optional: true,
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryMachineLearning,
		"grafana_machine_learning_holiday",
		resourceHolidayID,
		schema,
	).
		WithLister(lister(listHolidays)).
		WithPreferredResourceNameField("name")
}

func listHolidays(ctx context.Context, client *mlapi.Client) ([]string, error) {
	holidays, err := client.Holidays(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(holidays))
	for i, holiday := range holidays {
		ids[i] = holiday.ID
	}
	return ids, nil
}

func resourceHolidayCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	holiday, err := makeMLHoliday(d)
	if err != nil {
		return diag.FromErr(err)
	}
	holiday, err = c.NewHoliday(ctx, holiday)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(holiday.ID)
	return resourceHolidayRead(ctx, d, meta)
}

func resourceHolidayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	holiday, err := c.Holiday(ctx, d.Id())
	if err, shouldReturn := common.CheckReadError("holiday", d, err); shouldReturn {
		return err
	}

	customPeriods := make([]interface{}, 0, len(holiday.CustomPeriods))
	for _, cp := range holiday.CustomPeriods {
		p := map[string]interface{}{
			"name":       cp.Name,
			"start_time": cp.StartTime.Format(time.RFC3339),
			"end_time":   cp.EndTime.Format(time.RFC3339),
		}
		customPeriods = append(customPeriods, p)
	}

	d.Set("name", holiday.Name)
	d.Set("description", holiday.Description)
	d.Set("ical_url", holiday.ICalURL)
	d.Set("ical_timezone", holiday.ICalTimeZone)
	d.Set("custom_periods", customPeriods)

	return nil
}

func resourceHolidayUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	job, err := makeMLHoliday(d)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.UpdateHoliday(ctx, job)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceHolidayRead(ctx, d, meta)
}

func resourceHolidayDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	err := c.DeleteHoliday(ctx, d.Id())
	return diag.FromErr(err)
}

func makeMLHoliday(d *schema.ResourceData) (mlapi.Holiday, error) {
	cp := d.Get("custom_periods").([]interface{})
	customPeriods := make([]mlapi.CustomPeriod, 0, len(cp))
	for _, p := range cp {
		p := p.(map[string]interface{})
		startTime, err := time.Parse(time.RFC3339, p["start_time"].(string))
		if err != nil {
			return mlapi.Holiday{}, fmt.Errorf("failed to parse start_time %s: %w", p["start_time"], err)
		}
		endTime, err := time.Parse(time.RFC3339, p["end_time"].(string))
		if err != nil {
			return mlapi.Holiday{}, fmt.Errorf("failed to parse end_time %s: %w", p["end_time"].(string), err)
		}
		customPeriods = append(customPeriods, mlapi.CustomPeriod{
			Name:      p["name"].(string),
			StartTime: startTime,
			EndTime:   endTime,
		})
	}
	var iCalURL *string
	var iCalTimeZone *string
	if i := d.Get("ical_url").(string); i != "" {
		iCalURL = &i
	}
	if i := d.Get("ical_timezone").(string); i != "" {
		iCalTimeZone = &i
	}
	return mlapi.Holiday{
		ID:            d.Id(),
		Name:          d.Get("name").(string),
		Description:   d.Get("description").(string),
		ICalURL:       iCalURL,
		ICalTimeZone:  iCalTimeZone,
		CustomPeriods: customPeriods,
	}, nil
}
