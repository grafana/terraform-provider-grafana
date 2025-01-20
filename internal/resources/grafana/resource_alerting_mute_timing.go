package grafana

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceMuteTiming() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Alerting mute timings.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#mute-timings)

This resource requires Grafana 9.1.0 or later.
`,

		CreateContext: client.WithAlertingMutex[schema.CreateContextFunc](createMuteTiming),
		ReadContext:   readMuteTiming,
		UpdateContext: client.WithAlertingMutex[schema.UpdateContextFunc](updateMuteTiming),
		DeleteContext: client.WithAlertingMutex[schema.DeleteContextFunc](deleteMuteTiming),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the mute timing.",
			},
			"disable_provenance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true, // TODO: The API doesn't return provenance, so we have to force new for now.
				Description: "Allow modifying the mute timing from other sources than Terraform or the Grafana API.",
			},

			"intervals": {
				// List instead of set is necessary here. We rely on diff-suppression on the `months` field.
				// TF represents sets internally as dics, with hashes as keys.
				// If we use a set, the object hash is different any time a nested object gets changed.
				// Therefore TF will see delete+create instead of modify, which breaks the diff-suppression.
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The time intervals at which to mute notifications. Use an empty block to mute all the time.",
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
						"location": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: `Provides the time zone for the time interval. Must be a location in the IANA time zone database, e.g "America/New_York"`,
						},
					},
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_mute_timing",
		orgResourceIDString("name"),
		schema,
	).WithLister(listerFunctionOrgResource(listMuteTimings))
}

func listMuteTimings(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	// Retry if the API returns 500 because it may be that the alertmanager is not ready in the org yet.
	// The alertmanager is provisioned asynchronously when the org is created.
	if err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.GetMuteTimings()
		if err != nil {
			if orgID > 1 && (err.(*runtime.APIError).IsCode(500) || err.(*runtime.APIError).IsCode(403)) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		for _, muteTiming := range resp.Payload {
			ids = append(ids, MakeOrgResourceID(orgID, muteTiming.Name))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return ids, nil
}

func readMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	resp, err := client.Provisioning.GetMuteTiming(name)
	if err, shouldReturn := common.CheckReadError("mute timing", data, err); shouldReturn {
		return err
	}
	mt := resp.Payload

	data.SetId(MakeOrgResourceID(orgID, mt.Name))
	data.Set("org_id", strconv.FormatInt(orgID, 10))
	data.Set("name", mt.Name)
	data.Set("intervals", packIntervals(mt.TimeIntervals))
	return nil
}

func createMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	intervals := data.Get("intervals").([]interface{})
	params := provisioning.NewPostMuteTimingParams().
		WithBody(&models.MuteTimeInterval{
			Name:          data.Get("name").(string),
			TimeIntervals: unpackIntervals(intervals),
		})

	if v, ok := data.GetOk("disable_provenance"); ok && v.(bool) {
		params.SetXDisableProvenance(&provenanceDisabled)
	}

	var resp *provisioning.PostMuteTimingCreated
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		var postErr error
		resp, postErr = client.Provisioning.PostMuteTiming(params)
		if orgID > 1 && postErr != nil {
			if apiError, ok := postErr.(*runtime.APIError); ok && (apiError.IsCode(500) || apiError.IsCode(404)) {
				return retry.RetryableError(postErr)
			}
		}
		if postErr != nil {
			return retry.NonRetryableError(postErr)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(MakeOrgResourceID(orgID, resp.Payload.Name))
	return readMuteTiming(ctx, data, meta)
}

func updateMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	intervals := data.Get("intervals").([]interface{})
	params := provisioning.NewPutMuteTimingParams().
		WithName(name).
		WithBody(&models.MuteTimeInterval{
			Name:          name,
			TimeIntervals: unpackIntervals(intervals),
		})

	if v, ok := data.GetOk("disable_provenance"); ok && v.(bool) {
		params.SetXDisableProvenance(&provenanceDisabled)
	}

	_, err := client.Provisioning.PutMuteTiming(params)
	if err != nil {
		return diag.FromErr(err)
	}
	return readMuteTiming(ctx, data, meta)
}

func deleteMuteTiming(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	// Remove the mute timing from all notification policies
	policyResp, err := client.Provisioning.GetPolicyTree()
	if err != nil {
		return diag.FromErr(err)
	}
	policy := policyResp.Payload
	modified := false
	policy, modified = removeMuteTimingFromRoute(name, policy)
	if modified {
		_, err = client.Provisioning.PutPolicyTree(provisioning.NewPutPolicyTreeParams().WithBody(policy))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Remove the mute timing from alert rules
	ruleResp, err := client.Provisioning.GetAlertRules()
	if err != nil {
		return diag.FromErr(err)
	}
	rules := ruleResp.Payload
	for _, rule := range rules {
		if rule.NotificationSettings == nil {
			continue
		}

		var muteTimeIntervals []string
		for _, m := range rule.NotificationSettings.MuteTimeIntervals {
			if m != name {
				muteTimeIntervals = append(muteTimeIntervals, m)
			}
		}
		if len(muteTimeIntervals) != len(rule.NotificationSettings.MuteTimeIntervals) {
			rule.NotificationSettings.MuteTimeIntervals = muteTimeIntervals
			params := provisioning.NewPutAlertRuleParams().WithBody(rule).WithUID(rule.UID)
			_, err = client.Provisioning.PutAlertRule(params)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	// Delete the mute timing
	params := provisioning.NewDeleteMuteTimingParams().WithName(name)
	_, err = client.Provisioning.DeleteMuteTiming(params)
	diag, _ := common.CheckReadError("mute timing", data, err)
	return diag
}

func removeMuteTimingFromRoute(name string, route *models.Route) (*models.Route, bool) {
	modified := false
	for i, m := range route.MuteTimeIntervals {
		if m == name {
			route.MuteTimeIntervals = append(route.MuteTimeIntervals[:i], route.MuteTimeIntervals[i+1:]...)
			modified = true
			break
		}
	}
	for j, p := range route.Routes {
		var subRouteModified bool
		route.Routes[j], subRouteModified = removeMuteTimingFromRoute(name, p)
		modified = modified || subRouteModified
	}

	return route, modified
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

func packIntervals(nts []*models.TimeIntervalItem) []interface{} {
	if nts == nil {
		return nil
	}

	intervals := make([]interface{}, 0, len(nts))
	for _, ti := range nts {
		in := map[string]interface{}{}
		if ti.Times != nil {
			times := make([]interface{}, 0, len(ti.Times))
			for _, time := range ti.Times {
				times = append(times, packTimeRange(time))
			}
			in["times"] = times
		}
		if ti.Weekdays != nil {
			in["weekdays"] = common.StringSliceToList(ti.Weekdays)
		}
		if ti.DaysOfMonth != nil {
			in["days_of_month"] = common.StringSliceToList(ti.DaysOfMonth)
		}
		if ti.Months != nil {
			in["months"] = common.StringSliceToList(ti.Months)
		}
		if ti.Years != nil {
			in["years"] = common.StringSliceToList(ti.Years)
		}
		if ti.Location != "" {
			in["location"] = ti.Location
		}
		intervals = append(intervals, in)
	}

	return intervals
}

func unpackIntervals(raw []interface{}) []*models.TimeIntervalItem {
	if raw == nil {
		return nil
	}

	result := make([]*models.TimeIntervalItem, len(raw))
	for i, r := range raw {
		interval := models.TimeIntervalItem{}

		block := map[string]interface{}{}
		if r != nil {
			block = r.(map[string]interface{})
		}

		if vals, ok := block["times"]; ok && vals != nil {
			vals := vals.([]interface{})
			interval.Times = make([]*models.TimeIntervalTimeRange, len(vals))
			for i := range vals {
				interval.Times[i] = unpackTimeRange(vals[i])
			}
		}
		if vals, ok := block["weekdays"]; ok && vals != nil {
			interval.Weekdays = common.ListToStringSlice(vals.([]interface{}))
		}
		if vals, ok := block["days_of_month"]; ok && vals != nil {
			interval.DaysOfMonth = common.ListToStringSlice(vals.([]interface{}))
		}
		if vals, ok := block["months"]; ok && vals != nil {
			interval.Months = common.ListToStringSlice(vals.([]interface{}))
		}
		if vals, ok := block["years"]; ok && vals != nil {
			interval.Years = common.ListToStringSlice(vals.([]interface{}))
		}

		if vals, ok := block["location"]; ok && vals != nil {
			interval.Location = vals.(string)
		}

		result[i] = &interval
	}

	return result
}

func packTimeRange(time *models.TimeIntervalTimeRange) interface{} {
	return map[string]string{
		"start": time.StartTime,
		"end":   time.EndTime,
	}
}

func unpackTimeRange(raw interface{}) *models.TimeIntervalTimeRange {
	vals := raw.(map[string]interface{})
	return &models.TimeIntervalTimeRange{
		StartTime: vals["start"].(string),
		EndTime:   vals["end"].(string),
	}
}
