package grafana

import (
	"context"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceMuteTiming() *schema.Resource {
	return &schema.Resource{
		Description: `TODO`,

		ReadContext: readMuteTiming,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the mute timing.",
			},

			"intervals": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The time intervals at which to mute notifications.",
				Elem: &schema.Resource{
					SchemaVersion: 0,
					Schema: map[string]*schema.Schema{
						"times": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "The time ranges, represented in minutes, during which to mute in a given day.",
							Elem: &schema.Resource{
								SchemaVersion: 0,
								Schema: map[string]*schema.Schema{
									"start": {
										Type:        schema.TypeInt,
										Required:    true,
										Description: "The inclusive starting minute, within a 1440 minute day, of the time interval.",
									},
									"end": {
										Type:        schema.TypeInt,
										Required:    true,
										Description: "The exclusive ending minute, within a 1440 minute day, of the time interval.",
									},
								},
							},
						},
						"weekdays": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: `An inclusive range of weekdays, e.g. "monday" or "tuesday:thursday".`,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"days_of_month": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: `An inclusive range of days, 1-31, within a month, e.g. "1" or "14:16". Negative values can be used to represent days counting from the end of a month, e.g. "-1".`,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"months": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: `An inclusive range of months, either numerical or full calendar month, e.g. "1:3", "december", or "may:august".`,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"years": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: `A positive inclusive range of years, e.g. "2030" or "2025:2026".`,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func readMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	name := data.Id()
	mt, err := client.MuteTiming(name)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing mute timing %s from state because it no longer exists in grafana", name)
			data.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	data.SetId(mt.Name)
	data.Set("name", mt.Name)
	if mt.TimeIntervals != nil {
		intervals := make([]interface{}, 0)
		for _, ti := range mt.TimeIntervals {
			in := map[string][]interface{}{}
			if ti.Times != nil {
				times := []interface{}{}
				for _, time := range ti.Times {
					times = append(times, map[string]int{
						"start": time.StartMinute,
						"end":   time.EndMinute,
					})
				}
				in["times"] = []interface{}{}
			}
			if ti.Weekdays != nil {
				wkdays := make([]interface{}, 0)
				for _, wd := range ti.Weekdays {
					wkdays = append(wkdays, wd)
				}
				in["weekdays"] = wkdays
			}
			if ti.DaysOfMonth != nil {
				mdays := make([]interface{}, 0)
				for _, dom := range ti.DaysOfMonth {
					mdays = append(mdays, dom)
				}
				in["days_of_month"] = mdays
			}
			if ti.Months != nil {
				ms := make([]interface{}, 0)
				for _, m := range ti.Months {
					ms = append(ms, m)
				}
				in["months"] = ms
			}
			if ti.Years != nil {
				ys := make([]interface{}, 0)
				for _, y := range ti.Years {
					ys = append(ys, y)
				}
				in["years"] = ys
			}
			intervals = append(intervals, in)
		}
		data.Set("intervals", intervals)
	}

	return nil
}
