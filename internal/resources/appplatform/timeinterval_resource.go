package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var timeRangeType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"start_time": types.StringType,
		"end_time":   types.StringType,
	},
}

var intervalType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"times":         types.ListType{ElemType: timeRangeType},
		"weekdays":      types.ListType{ElemType: types.StringType},
		"days_of_month": types.ListType{ElemType: types.StringType},
		"months":        types.ListType{ElemType: types.StringType},
		"years":         types.ListType{ElemType: types.StringType},
		"location":      types.StringType,
	},
}

type timeIntervalTimeRangeModel struct {
	StartTime types.String `tfsdk:"start_time"`
	EndTime   types.String `tfsdk:"end_time"`
}

type timeIntervalIntervalModel struct {
	Times       types.List   `tfsdk:"times"`
	Weekdays    types.List   `tfsdk:"weekdays"`
	DaysOfMonth types.List   `tfsdk:"days_of_month"`
	Months      types.List   `tfsdk:"months"`
	Years       types.List   `tfsdk:"years"`
	Location    types.String `tfsdk:"location"`
}

type timeIntervalSpecModel struct {
	Name          types.String `tfsdk:"name"`
	TimeIntervals types.List   `tfsdk:"time_intervals"`
}

func TimeInterval() NamedResource {
	return NewNamedResource[*v1beta1.TimeInterval, *v1beta1.TimeIntervalList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.TimeInterval]{
			Kind:               v1beta1.TimeIntervalKind(),
			ServerGeneratedUID: true,
			Schema: ResourceSpecSchema{
				Description:         "Manages Grafana Time Intervals.",
				MarkdownDescription: "Manages Grafana Time Intervals.",
				SpecAttributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the time interval.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
				SpecBlocks: map[string]schema.Block{
					"time_intervals": schema.ListNestedBlock{
						Description: "A list of time interval definitions.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"times": schema.ListAttribute{
									Optional:    true,
									ElementType: timeRangeType,
									Description: "Ranges of time within a day.",
								},
								"weekdays": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "Days of the week (e.g. monday, tuesday).",
								},
								"days_of_month": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "Days of the month (1-31, negative values count from end of month).",
								},
								"months": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "Months of the year (e.g. january, february).",
								},
								"years": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "Calendar years (e.g. 2024, 2025:2027).",
								},
								"location": schema.StringAttribute{
									Optional:    true,
									Description: "IANA timezone name (e.g. America/New_York). Defaults to UTC when unset.",
								},
							},
						},
					},
				},
			},
			SpecParser: parseTimeIntervalSpec,
			SpecSaver:  saveTimeIntervalSpec,
		})
}

func parseTimeIntervalSpec(ctx context.Context, src types.Object, dst *v1beta1.TimeInterval) diag.Diagnostics {
	var data timeIntervalSpecModel
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	spec := v1beta1.TimeIntervalSpec{
		Name: data.Name.ValueString(),
	}

	if !data.TimeIntervals.IsNull() && !data.TimeIntervals.IsUnknown() {
		intervals, diags := parseTimeIntervalIntervals(ctx, data.TimeIntervals)
		if diags.HasError() {
			return diags
		}
		spec.TimeIntervals = intervals
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to get metadata accessor", err.Error()),
		}
	}
	meta.SetAnnotation(v1beta1.ProvenanceStatusAnnotationKey, provenanceAPI)

	return diag.Diagnostics{}
}

func parseTimeIntervalIntervals(ctx context.Context, src types.List) ([]v1beta1.TimeIntervalInterval, diag.Diagnostics) {
	var models []timeIntervalIntervalModel
	if diags := src.ElementsAs(ctx, &models, false); diags.HasError() {
		return nil, diags
	}

	intervals := make([]v1beta1.TimeIntervalInterval, 0, len(models))
	for _, m := range models {
		interval := v1beta1.TimeIntervalInterval{}

		if !m.Times.IsNull() && !m.Times.IsUnknown() {
			var timeRangeModels []timeIntervalTimeRangeModel
			if diags := m.Times.ElementsAs(ctx, &timeRangeModels, false); diags.HasError() {
				return nil, diags
			}
			times := make([]v1beta1.TimeIntervalTimeRange, 0, len(timeRangeModels))
			for _, tr := range timeRangeModels {
				times = append(times, v1beta1.TimeIntervalTimeRange{
					StartTime: tr.StartTime.ValueString(),
					EndTime:   tr.EndTime.ValueString(),
				})
			}
			interval.Times = times
		}

		if !m.Weekdays.IsNull() && !m.Weekdays.IsUnknown() {
			var weekdays []string
			if diags := m.Weekdays.ElementsAs(ctx, &weekdays, false); diags.HasError() {
				return nil, diags
			}
			interval.Weekdays = weekdays
		}

		if !m.DaysOfMonth.IsNull() && !m.DaysOfMonth.IsUnknown() {
			var daysOfMonth []string
			if diags := m.DaysOfMonth.ElementsAs(ctx, &daysOfMonth, false); diags.HasError() {
				return nil, diags
			}
			interval.DaysOfMonth = daysOfMonth
		}

		if !m.Months.IsNull() && !m.Months.IsUnknown() {
			var months []string
			if diags := m.Months.ElementsAs(ctx, &months, false); diags.HasError() {
				return nil, diags
			}
			interval.Months = months
		}

		if !m.Years.IsNull() && !m.Years.IsUnknown() {
			var years []string
			if diags := m.Years.ElementsAs(ctx, &years, false); diags.HasError() {
				return nil, diags
			}
			interval.Years = years
		}

		if !m.Location.IsNull() && !m.Location.IsUnknown() {
			loc := m.Location.ValueString()
			interval.Location = &loc
		}

		intervals = append(intervals, interval)
	}
	return intervals, nil
}

func saveTimeIntervalSpec(ctx context.Context, src *v1beta1.TimeInterval, dst *ResourceModel) diag.Diagnostics {
	intervals, diags := timeIntervalIntervalsToTf(ctx, src.Spec.TimeIntervals)
	if diags.HasError() {
		return diags
	}

	spec, diags := types.ObjectValue(
		map[string]attr.Type{
			"name":           types.StringType,
			"time_intervals": types.ListType{ElemType: intervalType},
		},
		map[string]attr.Value{
			"name":           types.StringValue(src.Spec.Name),
			"time_intervals": intervals,
		},
	)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec
	return diag.Diagnostics{}
}

func timeIntervalIntervalsToTf(ctx context.Context, intervals []v1beta1.TimeIntervalInterval) (types.List, diag.Diagnostics) {
	models := make([]timeIntervalIntervalModel, 0, len(intervals))
	for _, interval := range intervals {
		m := timeIntervalIntervalModel{}

		if len(interval.Times) > 0 {
			timeRangeModels := make([]timeIntervalTimeRangeModel, 0, len(interval.Times))
			for _, tr := range interval.Times {
				timeRangeModels = append(timeRangeModels, timeIntervalTimeRangeModel{
					StartTime: types.StringValue(tr.StartTime),
					EndTime:   types.StringValue(tr.EndTime),
				})
			}
			times, diags := types.ListValueFrom(ctx, timeRangeType, timeRangeModels)
			if diags.HasError() {
				return types.ListNull(intervalType), diags
			}
			m.Times = times
		} else {
			m.Times = types.ListNull(timeRangeType)
		}

		if len(interval.Weekdays) > 0 {
			weekdays, diags := types.ListValueFrom(ctx, types.StringType, interval.Weekdays)
			if diags.HasError() {
				return types.ListNull(intervalType), diags
			}
			m.Weekdays = weekdays
		} else {
			m.Weekdays = types.ListNull(types.StringType)
		}

		if len(interval.DaysOfMonth) > 0 {
			daysOfMonth, diags := types.ListValueFrom(ctx, types.StringType, interval.DaysOfMonth)
			if diags.HasError() {
				return types.ListNull(intervalType), diags
			}
			m.DaysOfMonth = daysOfMonth
		} else {
			m.DaysOfMonth = types.ListNull(types.StringType)
		}

		if len(interval.Months) > 0 {
			months, diags := types.ListValueFrom(ctx, types.StringType, interval.Months)
			if diags.HasError() {
				return types.ListNull(intervalType), diags
			}
			m.Months = months
		} else {
			m.Months = types.ListNull(types.StringType)
		}

		if len(interval.Years) > 0 {
			years, diags := types.ListValueFrom(ctx, types.StringType, interval.Years)
			if diags.HasError() {
				return types.ListNull(intervalType), diags
			}
			m.Years = years
		} else {
			m.Years = types.ListNull(types.StringType)
		}

		if interval.Location != nil {
			m.Location = types.StringValue(*interval.Location)
		} else {
			m.Location = types.StringNull()
		}

		models = append(models, m)
	}
	return types.ListValueFrom(ctx, intervalType, models)
}
