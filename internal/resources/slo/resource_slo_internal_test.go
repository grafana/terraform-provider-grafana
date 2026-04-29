package slo

import (
	"context"
	"testing"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// TestUnit_convertQueryToModel_ratioGroupByLabels covers the read side of the
// empty-group_by_labels round-trip:
//
// The SLO API marshals optional fields with `omitempty`, so a PUT of
// `{groupByLabels: []}` round-trips to a GET response with the field absent,
// which the OpenAPI client decodes as a nil slice. Without an explicit nil
// promotion, types.ListValueFrom(ctx, StringType, nil) produced a nil
// types.List while a user HCL value of `group_by_labels = []` plans as an
// empty list, triggering terraform's:
//
//	Provider produced inconsistent result after apply: .query[0].ratio[0].group_by_labels:
//	was cty.ListValEmpty(cty.String), but now null.
//
// The fix promotes both nil and empty API responses to a non-nil empty list;
// the EmptyListForNullConfig plan modifier (tested separately) brings nil
// configs to the same shape so plan and state agree.
func TestUnit_convertQueryToModel_ratioGroupByLabels(t *testing.T) {
	cases := []struct {
		name         string
		apiGroupBy   []string
		wantNull     bool
		wantElements int
	}{
		{"nil_slice_simulates_omitted_field", nil, false, 0},
		{"empty_slice_explicit", []string{}, false, 0},
		{"populated_slice", []string{"job", "instance"}, false, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiQuery := slo.SloV00Query{
				Type: QueryTypeRatio,
				Ratio: &slo.SloV00RatioQuery{
					SuccessMetric: slo.SloV00MetricDef{PrometheusMetric: "success_total"},
					TotalMetric:   slo.SloV00MetricDef{PrometheusMetric: "total"},
					GroupByLabels: tc.apiGroupBy,
				},
			}

			models, diags := convertQueryToModel(context.Background(), apiQuery)
			require.False(t, diags.HasError(), "convertQueryToModel diags: %v", diags)
			require.Len(t, models, 1)
			require.Len(t, models[0].Ratio, 1)

			got := models[0].Ratio[0].GroupByLabels
			require.Equalf(t, tc.wantNull, got.IsNull(),
				"IsNull mismatch for %s: %s", tc.name, got.String())
			require.Equal(t, tc.wantElements, len(got.Elements()))
		})
	}
}

// TestUnit_emptyListForNullConfig_planModifier covers the config-side half of
// the fix: a nil config value is rewritten to an empty list, while empty and
// populated configs are passed through unchanged. terraform Core's
// plan-validity rule requires `plan == config` for non-nil config values
// (even with `Computed: true`), so we can only modify the plan when config
// itself is nil.
func TestUnit_emptyListForNullConfig_planModifier(t *testing.T) {
	ctx := context.Background()

	emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
	populated, _ := types.ListValueFrom(ctx, types.StringType, []string{"job"})

	cases := []struct {
		name        string
		config      types.List
		initialPlan types.List
		wantPlan    types.List
	}{
		{
			"nil_config_becomes_empty_list",
			types.ListNull(types.StringType), types.ListUnknown(types.StringType),
			emptyList,
		},
		{
			"empty_config_unchanged",
			emptyList, emptyList,
			emptyList,
		},
		{
			"populated_config_unchanged",
			populated, populated,
			populated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.ListRequest{
				ConfigValue: tc.config,
				PlanValue:   tc.initialPlan,
			}
			resp := planmodifier.ListResponse{PlanValue: tc.initialPlan}

			EmptyListForNullConfig().PlanModifyList(ctx, req, &resp)

			require.False(t, resp.Diagnostics.HasError(), "modifier diags: %v", resp.Diagnostics)
			require.True(t, tc.wantPlan.Equal(resp.PlanValue),
				"plan mismatch: want %s got %s", tc.wantPlan.String(), resp.PlanValue.String())
		})
	}
}

// TestUnit_groupByLabels_roundTripIsConsistent ties the two halves together.
// All three HCL forms — populated, `[]`, omitted — must produce a non-nil
// empty-or-populated state value that compares equal to the resolved plan,
// otherwise terraform raises the inconsistent-result error. The forward
// `pack` path drops empty lists at omitempty serialization time, so the API
// stores nothing and GET returns nil; the backward `convertQueryToModel`
// promotes that nil to `[]` and the schema's plan modifier mirrors it for
// nil config.
func TestUnit_groupByLabels_roundTripIsConsistent(t *testing.T) {
	ctx := context.Background()

	emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})

	// Case A: user HCL has `group_by_labels = []`. Plan is `[]` (no modifier
	// touch since config is non-nil), API drops the field, read returns `[]`.
	{
		req := planmodifier.ListRequest{ConfigValue: emptyList, PlanValue: emptyList}
		resp := planmodifier.ListResponse{PlanValue: emptyList}
		EmptyListForNullConfig().PlanModifyList(ctx, req, &resp)
		planValue := resp.PlanValue
		require.True(t, emptyList.Equal(planValue), "explicit empty config: plan should remain empty list")

		apiQuery := slo.SloV00Query{
			Type: QueryTypeRatio,
			Ratio: &slo.SloV00RatioQuery{
				SuccessMetric: slo.SloV00MetricDef{PrometheusMetric: "success_total"},
				TotalMetric:   slo.SloV00MetricDef{PrometheusMetric: "total"},
				GroupByLabels: nil,
			},
		}
		models, diags := convertQueryToModel(ctx, apiQuery)
		require.False(t, diags.HasError())
		stateValue := models[0].Ratio[0].GroupByLabels

		require.True(t, planValue.Equal(stateValue),
			"explicit-empty round-trip: plan %s != state %s", planValue.String(), stateValue.String())
	}

	// Case B: user HCL omits the attribute. ConfigValue is nil, plan modifier
	// rewrites to `[]`, API returns nothing, read returns `[]`.
	{
		nullConfig := types.ListNull(types.StringType)
		req := planmodifier.ListRequest{ConfigValue: nullConfig, PlanValue: types.ListUnknown(types.StringType)}
		resp := planmodifier.ListResponse{PlanValue: types.ListUnknown(types.StringType)}
		EmptyListForNullConfig().PlanModifyList(ctx, req, &resp)
		planValue := resp.PlanValue
		require.True(t, emptyList.Equal(planValue), "nil config: plan should be empty list after modifier")

		apiQuery := slo.SloV00Query{
			Type: QueryTypeRatio,
			Ratio: &slo.SloV00RatioQuery{
				SuccessMetric: slo.SloV00MetricDef{PrometheusMetric: "success_total"},
				TotalMetric:   slo.SloV00MetricDef{PrometheusMetric: "total"},
				GroupByLabels: nil,
			},
		}
		models, diags := convertQueryToModel(ctx, apiQuery)
		require.False(t, diags.HasError())
		stateValue := models[0].Ratio[0].GroupByLabels

		require.True(t, planValue.Equal(stateValue),
			"omitted-attribute round-trip: plan %s != state %s", planValue.String(), stateValue.String())
	}
}
