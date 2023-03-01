package machinelearning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceHoliday() *schema.Resource {
	return &schema.Resource{

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

		CreateContext: ResourceHolidayCreate,
		ReadContext:   ResourceHolidayRead,
		UpdateContext: ResourceHolidayUpdate,
		DeleteContext: ResourceHolidayDelete,
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
}

func ResourceHolidayCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	return ResourceHolidayRead(ctx, d, meta)
}

func ResourceHolidayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	holiday, err := c.Holiday(ctx, d.Id())
	if err != nil {
		var diags diag.Diagnostics
		if strings.HasPrefix(err.Error(), "status: 404") {
			name := d.Get("name").(string)
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Holiday %q is in Terraform state, but no longer exists in Grafana ML", name),
				Detail:   fmt.Sprintf("%q will be recreated when you apply", name),
			})
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
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

func ResourceHolidayUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	job, err := makeMLHoliday(d)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.UpdateHoliday(ctx, job)
	if err != nil {
		return diag.FromErr(err)
	}
	return ResourceHolidayRead(ctx, d, meta)
}

func ResourceHolidayDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	err := c.DeleteHoliday(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
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
