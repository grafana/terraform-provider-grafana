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
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

var (
	_ resource.Resource                = &muteTimingResource{}
	_ resource.ResourceWithConfigure   = &muteTimingResource{}
	_ resource.ResourceWithImportState = &muteTimingResource{}

	resourceMuteTimingName = "grafana_mute_timing"
	resourceMuteTimingID   = orgResourceIDString("name")
)

func makeResourceMuteTiming() *common.Resource {
	return common.NewResource(
		common.CategoryAlerting,
		resourceMuteTimingName,
		resourceMuteTimingID,
		&muteTimingResource{},
	).WithLister(listerFunctionOrgResource(listMuteTimings))
}

type muteTimingModel struct {
	ID                types.String              `tfsdk:"id"`
	OrgID             types.String              `tfsdk:"org_id"`
	Name              types.String              `tfsdk:"name"`
	DisableProvenance types.Bool                `tfsdk:"disable_provenance"`
	Intervals         []muteTimingIntervalModel `tfsdk:"intervals"`
}

type muteTimingIntervalModel struct {
	Times       []muteTimingTimeRangeModel `tfsdk:"times"`
	Weekdays    types.List                 `tfsdk:"weekdays"`
	DaysOfMonth types.List                 `tfsdk:"days_of_month"`
	Months      types.List                 `tfsdk:"months"`
	Years       types.List                 `tfsdk:"years"`
	Location    types.String               `tfsdk:"location"`
}

type muteTimingTimeRangeModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type muteTimingResource struct {
	basePluginFrameworkResource
}

func (r *muteTimingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceMuteTimingName
}

func (r *muteTimingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages Grafana Alerting mute timings.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#mute-timings)

This resource requires Grafana 9.1.0 or later.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the mute timing.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disable_provenance": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow modifying the mute timing from other sources than Terraform or the Grafana API. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
				// TODO: The API doesn't return provenance, so we have to force new for now.
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"intervals": schema.ListNestedBlock{
				Description: "The time intervals at which to mute notifications. Use an empty block to mute all the time.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"weekdays": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: `An inclusive range of weekdays, e.g. "monday" or "tuesday:thursday".`,
						},
						"days_of_month": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: `An inclusive range of days, 1-31, within a month, e.g. "1" or "14:16". Negative values can be used to represent days counting from the end of a month, e.g. "-1".`,
						},
						"months": schema.ListAttribute{
							Optional:    true,
							ElementType: monthStringType{},
							Description: `An inclusive range of months, either numerical or full calendar month, e.g. "1:3", "december", or "may:august".`,
						},
						"years": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: `A positive inclusive range of years, e.g. "2030" or "2025:2026".`,
						},
						"location": schema.StringAttribute{
							Optional:    true,
							Description: `Provides the time zone for the time interval. Must be a location in the IANA time zone database, e.g "America/New_York"`,
						},
					},
					Blocks: map[string]schema.Block{
						"times": schema.ListNestedBlock{
							Description: "The time ranges, represented in minutes, during which to mute in a given day.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"start": schema.StringAttribute{
										Required:    true,
										Description: "The time, in hh:mm format, of when the interval should begin inclusively.",
									},
									"end": schema.StringAttribute{
										Required:    true,
										Description: "The time, in hh:mm format, of when the interval should end exclusively.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *muteTimingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan muteTimingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	intervals, diags := muteTimingModelToIntervals(ctx, plan.Intervals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.NewPostMuteTimingParams().
		WithBody(&models.MuteTimeInterval{
			Name:          plan.Name.ValueString(),
			TimeIntervals: intervals,
		})
	if plan.DisableProvenance.ValueBool() {
		params.SetXDisableProvenance(&provenanceDisabled)
	}

	var name string
	var createErr error
	r.commonClient.WithAlertingLock(func() {
		createErr = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
			postResp, postErr := client.Provisioning.PostMuteTiming(params)
			if postErr != nil {
				if clientErr, ok := postErr.(runtime.ClientResponseStatus); ok && (clientErr.IsCode(500) || clientErr.IsCode(403)) {
					return retry.RetryableError(postErr)
				}
				return retry.NonRetryableError(postErr)
			}
			name = postResp.Payload.Name
			return nil
		})
	})
	if createErr != nil {
		resp.Diagnostics.AddError("Failed to create mute timing", createErr.Error())
		return
	}

	plan.ID = types.StringValue(MakeOrgResourceID(orgID, name))
	readData, diags := r.read(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readData.DisableProvenance = plan.DisableProvenance
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *muteTimingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state muteTimingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	// The API doesn't return provenance; preserve the value from state.
	readData.DisableProvenance = state.DisableProvenance
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *muteTimingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan muteTimingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceMuteTimingID, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	name := split[0].(string)

	intervals, diags := muteTimingModelToIntervals(ctx, plan.Intervals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.NewPutMuteTimingParams().
		WithName(name).
		WithBody(&models.MuteTimeInterval{
			Name:          name,
			TimeIntervals: intervals,
		})
	if plan.DisableProvenance.ValueBool() {
		params.SetXDisableProvenance(&provenanceDisabled)
	}

	var updateErr error
	r.commonClient.WithAlertingLock(func() {
		_, updateErr = client.Provisioning.PutMuteTiming(params)
	})
	if updateErr != nil {
		resp.Diagnostics.AddError("Failed to update mute timing", updateErr.Error())
		return
	}

	readData, diags := r.read(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readData.DisableProvenance = plan.DisableProvenance
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *muteTimingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state muteTimingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceMuteTimingID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	name := split[0].(string)

	var deleteErr error
	r.commonClient.WithAlertingLock(func() {
		// Remove the mute timing from all notification policies.
		policyResp, err := client.Provisioning.GetPolicyTree()
		if err != nil {
			deleteErr = err
			return
		}
		policy := policyResp.Payload
		var modified bool
		policy, modified = removeMuteTimingFromRoute(name, policy)
		if modified {
			_, err = client.Provisioning.PutPolicyTree(provisioning.NewPutPolicyTreeParams().WithBody(policy))
			if err != nil {
				deleteErr = err
				return
			}
		}

		// Remove the mute timing from alert rules.
		ruleResp, err := client.Provisioning.GetAlertRules()
		if err != nil {
			deleteErr = err
			return
		}
		for _, rule := range ruleResp.Payload {
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
					deleteErr = err
					return
				}
			}
		}

		// Delete the mute timing itself.
		params := provisioning.NewDeleteMuteTimingParams().WithName(name)
		_, deleteErr = client.Provisioning.DeleteMuteTiming(params)
	})

	if deleteErr != nil && !common.IsNotFoundError(deleteErr) {
		resp.Diagnostics.AddError("Failed to delete mute timing", deleteErr.Error())
	}
}

func (r *muteTimingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Mute timing not found")
		return
	}
	// The API doesn't return provenance; default to false on import.
	readData.DisableProvenance = types.BoolValue(false)
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *muteTimingResource) read(ctx context.Context, id string) (*muteTimingModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, orgID, split, err := r.clientFromExistingOrgResource(resourceMuteTimingID, id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, diags
	}
	name := split[0].(string)

	resp, err := client.Provisioning.GetMuteTiming(name)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Failed to read mute timing", err.Error())
		return nil, diags
	}
	mt := resp.Payload

	intervals, d := muteTimingAPIToModel(ctx, mt.TimeIntervals)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	return &muteTimingModel{
		ID:        types.StringValue(MakeOrgResourceID(orgID, mt.Name)),
		OrgID:     types.StringValue(strconv.FormatInt(orgID, 10)),
		Name:      types.StringValue(mt.Name),
		Intervals: intervals,
		// DisableProvenance is set by the caller; the API does not return it.
	}, diags
}

func listMuteTimings(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	// Retry if the API returns 500 because the alertmanager may not be ready yet.
	// The alertmanager is provisioned asynchronously when an org is created.
	if err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.GetMuteTimings()
		if err != nil {
			if clientErr, ok := err.(runtime.ClientResponseStatus); ok && (clientErr.IsCode(500) || clientErr.IsCode(403)) {
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

// removeMuteTimingFromRoute removes all references to the named mute timing from a policy route tree.
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

// muteTimingMonthNumbers maps month names to their numeric equivalents for normalization.
var muteTimingMonthNumbers = map[string]string{
	"january": "1", "february": "2", "march": "3", "april": "4",
	"may": "5", "june": "6", "july": "7", "august": "8",
	"september": "9", "october": "10", "november": "11", "december": "12",
}

// normalizeMonthToNumber converts a month value or month range to numeric form.
// E.g. "december" → "12", "january:march" → "1:3". Used only for semantic comparison.
func normalizeMonthToNumber(m string) string {
	m = strings.ToLower(m)
	for name, num := range muteTimingMonthNumbers {
		m = strings.ReplaceAll(m, name, num)
	}
	return m
}

// monthStringType is a custom Plugin Framework string type for month values.
// Its value type (monthStringValue) implements StringSemanticEquals so that numeric and
// name forms are considered equal (e.g. "december" == "12"), replacing the SDKv2 DiffSuppressFunc.
type monthStringType struct {
	basetypes.StringType
}

func (t monthStringType) Equal(o attr.Type) bool { _, ok := o.(monthStringType); return ok }
func (t monthStringType) String() string         { return "monthStringType" }
func (t monthStringType) ValueType(_ context.Context) attr.Value {
	return monthStringValue{}
}
func (t monthStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return monthStringValue{StringValue: in}, nil
}
func (t monthStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	sv, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.StringValue", attrValue)
	}
	return monthStringValue{StringValue: sv}, nil
}

// monthStringValue is the value type for monthStringType.
type monthStringValue struct {
	basetypes.StringValue
}

func (v monthStringValue) Type(_ context.Context) attr.Type { return monthStringType{} }
func (v monthStringValue) Equal(o attr.Value) bool {
	other, ok := o.(monthStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals returns true when the prior and proposed values are numerically equivalent
// months (e.g. "december" and "12"). This prevents Terraform from reporting a diff when the
// Grafana API returns a number but the config uses a name, matching the behaviour of the former
// SDKv2 DiffSuppressFunc.
func (v monthStringValue) StringSemanticEquals(_ context.Context, prior basetypes.StringValuable) (bool, diag.Diagnostics) {
	priorVal, ok := prior.(monthStringValue)
	if !ok {
		return false, nil
	}
	return normalizeMonthToNumber(v.ValueString()) == normalizeMonthToNumber(priorVal.ValueString()), nil
}

// muteTimingModelToIntervals converts Framework model structs to API interval structs.
func muteTimingModelToIntervals(ctx context.Context, intervals []muteTimingIntervalModel) ([]*models.TimeIntervalItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]*models.TimeIntervalItem, len(intervals))
	for i, interval := range intervals {
		item := &models.TimeIntervalItem{}

		for _, tr := range interval.Times {
			item.Times = append(item.Times, &models.TimeIntervalTimeRange{
				StartTime: tr.Start.ValueString(),
				EndTime:   tr.End.ValueString(),
			})
		}

		if !interval.Weekdays.IsNull() {
			var weekdays []string
			diags.Append(interval.Weekdays.ElementsAs(ctx, &weekdays, false)...)
			item.Weekdays = weekdays
		}
		if !interval.DaysOfMonth.IsNull() {
			var days []string
			diags.Append(interval.DaysOfMonth.ElementsAs(ctx, &days, false)...)
			item.DaysOfMonth = days
		}
		if !interval.Months.IsNull() {
			var months []string
			diags.Append(interval.Months.ElementsAs(ctx, &months, false)...)
			item.Months = months
		}
		if !interval.Years.IsNull() {
			var years []string
			diags.Append(interval.Years.ElementsAs(ctx, &years, false)...)
			item.Years = years
		}
		if !interval.Location.IsNull() {
			item.Location = interval.Location.ValueString()
		}

		result[i] = item
	}
	return result, diags
}

// muteTimingAPIToModel converts API interval structs to Framework model structs.
func muteTimingAPIToModel(ctx context.Context, nts []*models.TimeIntervalItem) ([]muteTimingIntervalModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(nts) == 0 {
		return nil, diags
	}

	intervals := make([]muteTimingIntervalModel, 0, len(nts))
	for _, ti := range nts {
		interval := muteTimingIntervalModel{
			Location:    types.StringNull(),
			Weekdays:    types.ListNull(types.StringType),
			DaysOfMonth: types.ListNull(types.StringType),
			Months:      types.ListNull(monthStringType{}),
			Years:       types.ListNull(types.StringType),
		}

		for _, t := range ti.Times {
			interval.Times = append(interval.Times, muteTimingTimeRangeModel{
				Start: types.StringValue(t.StartTime),
				End:   types.StringValue(t.EndTime),
			})
		}

		if ti.Weekdays != nil {
			v, d := types.ListValueFrom(ctx, types.StringType, ti.Weekdays)
			diags.Append(d...)
			interval.Weekdays = v
		}
		if ti.DaysOfMonth != nil {
			v, d := types.ListValueFrom(ctx, types.StringType, ti.DaysOfMonth)
			diags.Append(d...)
			interval.DaysOfMonth = v
		}
		if ti.Months != nil {
			elems := make([]attr.Value, len(ti.Months))
			for i, m := range ti.Months {
				elems[i] = monthStringValue{StringValue: basetypes.NewStringValue(m)}
			}
			v, d := types.ListValue(monthStringType{}, elems)
			diags.Append(d...)
			interval.Months = v
		}
		if ti.Years != nil {
			v, d := types.ListValueFrom(ctx, types.StringType, ti.Years)
			diags.Append(d...)
			interval.Years = v
		}
		if ti.Location != "" {
			interval.Location = types.StringValue(ti.Location)
		}

		intervals = append(intervals, interval)
	}
	return intervals, diags
}
