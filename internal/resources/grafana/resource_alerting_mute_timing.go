package grafana

import (
	"context"
	"fmt"
	"log"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceMuteTiming() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana Alerting mute timings.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/manage-notifications/mute-timings/)
* [HTTP API](https://grafana.com/docs/grafana/next/developers/http_api/alerting_provisioning/#mute-timings)

This resource requires Grafana 9.1.0 or later.
`,

		CreateContext: createMuteTiming,
		ReadContext:   readMuteTiming,
		UpdateContext: updateMuteTiming,
		DeleteContext: deleteMuteTiming,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the mute timing.",
			},

			"intervals": {
				// List instead of set is necessary here. We rely on diff-suppression on the `months` field.
				// TF represents sets internally as dics, with hashes as keys.
				// If we use a set, the object hash is different any time a nested object gets changed.
				// Therefore TF will see delete+create instead of modify, which breaks the diff-suppression.
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The time intervals at which to mute notifications.",
				Elem: &schema.Resource{
					SchemaVersion: 0,
					Schema: map[string]*schema.Schema{
						"times": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The time ranges, represented in minutes, during which to mute in a given day.",
							Elem: &schema.Resource{
								SchemaVersion: 0,
								Schema: map[string]*schema.Schema{
									"start": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The time, in hh:mm format, of when the interval should begin inclusively.",
									},
									"end": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The time, in hh:mm format, of when the interval should end exclusively.",
									},
								},
							},
						},
						"weekdays": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: `An inclusive range of weekdays, e.g. "monday" or "tuesday:thursday".`,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"days_of_month": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: `An inclusive range of days, 1-31, within a month, e.g. "1" or "14:16". Negative values can be used to represent days counting from the end of a month, e.g. "-1".`,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"months": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: `An inclusive range of months, either numerical or full calendar month, e.g. "1:3", "december", or "may:august".`,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							DiffSuppressFunc: suppressMonthDiff,
						},
						"years": {
							Type:        schema.TypeList,
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
	client := meta.(*common.Client).GrafanaAPI

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
	data.Set("intervals", packIntervals(mt.TimeIntervals))
	return nil
}

func createMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI

	mt := unpackMuteTiming(data)

	lock.Lock()
	defer lock.Unlock()
	if err := client.NewMuteTiming(&mt); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(mt.Name)
	return readMuteTiming(ctx, data, meta)
}

func updateMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI

	mt := unpackMuteTiming(data)

	lock.Lock()
	defer lock.Unlock()
	if err := client.UpdateMuteTiming(&mt); err != nil {
		return diag.FromErr(err)
	}
	return readMuteTiming(ctx, data, meta)
}

func deleteMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI
	name := data.Id()

	lock.Lock()
	defer lock.Unlock()
	if err := client.DeleteMuteTiming(name); err != nil {
		return diag.FromErr(err)
	}
	return diag.Diagnostics{}
}

func suppressMonthDiff(k, oldValue, newValue string, d *schema.ResourceData) bool {
	monthNums := map[string]int{
		"january":   1,
		"february":  2,
		"march":     3,
		"april":     4,
		"may":       5,
		"june":      6,
		"july":      7,
		"august":    8,
		"september": 9,
		"october":   10,
		"november":  11,
		"december":  12,
	}

	oldNormalized := oldValue
	newNormalized := newValue
	for k, v := range monthNums {
		oldNormalized = strings.ReplaceAll(oldNormalized, k, fmt.Sprint(v))
		newNormalized = strings.ReplaceAll(newNormalized, k, fmt.Sprint(v))
	}

	return oldNormalized == newNormalized
}

func unpackMuteTiming(d *schema.ResourceData) gapi.MuteTiming {
	intervals := d.Get("intervals").([]interface{})
	mt := gapi.MuteTiming{
		Name:          d.Get("name").(string),
		TimeIntervals: unpackIntervals(intervals),
	}
	return mt
}

func packIntervals(nts []gapi.TimeInterval) []interface{} {
	if nts == nil {
		return nil
	}

	intervals := make([]interface{}, 0, len(nts))
	for _, ti := range nts {
		in := map[string][]interface{}{}
		if ti.Times != nil {
			times := []interface{}{}
			for _, time := range ti.Times {
				times = append(times, packTimeRange(time))
			}
			in["times"] = times
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

	return intervals
}

func unpackIntervals(raw []interface{}) []gapi.TimeInterval {
	if raw == nil {
		return nil
	}

	result := make([]gapi.TimeInterval, len(raw))
	for i, r := range raw {
		interval := gapi.TimeInterval{}
		block := r.(map[string]interface{})

		if vals, ok := block["times"]; ok && vals != nil {
			vals := vals.([]interface{})
			interval.Times = make([]gapi.TimeRange, len(vals))
			for i := range vals {
				interval.Times[i] = unpackTimeRange(vals[i])
			}
		}
		if vals, ok := block["weekdays"]; ok && vals != nil {
			vals := vals.([]interface{})
			interval.Weekdays = make([]gapi.WeekdayRange, len(vals))
			for i := range vals {
				interval.Weekdays[i] = gapi.WeekdayRange(vals[i].(string))
			}
		}
		if vals, ok := block["days_of_month"]; ok && vals != nil {
			vals := vals.([]interface{})
			interval.DaysOfMonth = make([]gapi.DayOfMonthRange, len(vals))
			for i := range vals {
				interval.DaysOfMonth[i] = gapi.DayOfMonthRange(vals[i].(string))
			}
		}
		if vals, ok := block["months"]; ok && vals != nil {
			vals := vals.([]interface{})
			interval.Months = make([]gapi.MonthRange, len(vals))
			for i := range vals {
				interval.Months[i] = gapi.MonthRange(vals[i].(string))
			}
		}
		if vals, ok := block["years"]; ok && vals != nil {
			vals := vals.([]interface{})
			interval.Years = make([]gapi.YearRange, len(vals))
			for i := range vals {
				interval.Years[i] = gapi.YearRange(vals[i].(string))
			}
		}

		result[i] = interval
	}

	return result
}

func packTimeRange(time gapi.TimeRange) interface{} {
	return map[string]string{
		"start": time.StartMinute,
		"end":   time.EndMinute,
	}
}

func unpackTimeRange(raw interface{}) gapi.TimeRange {
	vals := raw.(map[string]interface{})
	return gapi.TimeRange{
		StartMinute: vals["start"].(string),
		EndMinute:   vals["end"].(string),
	}
}
