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
// check, types.ListValueFrom(ctx, StringType, nil) produced a *null* list
// while a user HCL block of `group_by_labels = []` planned an *empty* list,
// triggering terraform's:
//
//	Provider produced inconsistent result after apply: .query[0].ratio[0].group_by_labels:
//	was cty.ListValEmpty(cty.String), but now null.
//
// The fix collapses both nil and empty API responses to a null state value;
// the EmptyListAsNull plan modifier (tested separately) collapses the matching
// config shapes so plan and state agree.
func TestUnit_convertQueryToModel_ratioGroupByLabels(t *testing.T) {
	cases := []struct {
		name        string
		apiGroupBy  []string
		wantNull    bool
		wantElements int
	}{
		{"nil_slice_simulates_omitted_field", nil, true, 0},
		{"empty_slice_explicit", []string{}, true, 0},
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

// TestUnit_emptyListAsNull_planModifier exercises the config-side half of the
// fix: a known plan value of `[]` must be rewritten to null so it matches the
// null state value that convertQueryToModel produces for an API response with
// no groupByLabels. Null and unknown plan values pass through unchanged, and
// non-empty lists are not touched.
func TestUnit_emptyListAsNull_planModifier(t *testing.T) {
	ctx := context.Background()

	emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
	populated, _ := types.ListValueFrom(ctx, types.StringType, []string{"job"})

	cases := []struct {
		name      string
		in        types.List
		wantNull  bool
		wantEqual types.List
	}{
		{"empty_list_collapses_to_null", emptyList, true, types.ListNull(types.StringType)},
		{"null_passes_through", types.ListNull(types.StringType), true, types.ListNull(types.StringType)},
		{"unknown_passes_through", types.ListUnknown(types.StringType), false, types.ListUnknown(types.StringType)},
		{"populated_list_unchanged", populated, false, populated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.ListRequest{PlanValue: tc.in}
			resp := planmodifier.ListResponse{PlanValue: tc.in}

			EmptyListAsNull().PlanModifyList(ctx, req, &resp)

			require.Equalf(t, tc.wantNull, resp.PlanValue.IsNull(),
				"IsNull mismatch: got %s", resp.PlanValue.String())
			require.True(t, tc.wantEqual.Equal(resp.PlanValue),
				"value mismatch: want %s got %s", tc.wantEqual.String(), resp.PlanValue.String())
		})
	}
}

// TestUnit_groupByLabels_roundTripIsConsistent ties the two halves together:
// from a user HCL plan of `group_by_labels = []`, after the plan modifier
// normalizes it to null and the SLO API drops the field via omitempty,
// convertQueryToModel must produce a state value that compares equal to the
// modified plan — otherwise terraform raises the inconsistent-result error.
func TestUnit_groupByLabels_roundTripIsConsistent(t *testing.T) {
	ctx := context.Background()

	// 1. User HCL: group_by_labels = []  → framework presents as empty list.
	configList, _ := types.ListValueFrom(ctx, types.StringType, []string{})

	// 2. Plan modifier collapses `[]` → null.
	req := planmodifier.ListRequest{PlanValue: configList}
	resp := planmodifier.ListResponse{PlanValue: configList}
	EmptyListAsNull().PlanModifyList(ctx, req, &resp)
	planValue := resp.PlanValue
	require.True(t, planValue.IsNull(), "plan modifier should collapse empty list to null")

	// 3. Provider sends the null list to the API. Pack would also produce a
	//    nil slice for null; the API stores that, and the GET response omits
	//    the field — the OpenAPI client decodes that as a nil slice.
	apiQuery := slo.SloV00Query{
		Type: QueryTypeRatio,
		Ratio: &slo.SloV00RatioQuery{
			SuccessMetric: slo.SloV00MetricDef{PrometheusMetric: "success_total"},
			TotalMetric:   slo.SloV00MetricDef{PrometheusMetric: "total"},
			GroupByLabels: nil,
		},
	}

	// 4. Read converts the API response back to the framework model.
	models, diags := convertQueryToModel(ctx, apiQuery)
	require.False(t, diags.HasError(), "convertQueryToModel diags: %v", diags)
	stateValue := models[0].Ratio[0].GroupByLabels

	// 5. Round-trip must be consistent: state.Equal(plan) holds.
	require.True(t, planValue.Equal(stateValue),
		"round-trip lost shape: plan=%s state=%s — would trigger "+
			`"was cty.ListValEmpty(cty.String), but now null" inconsistency on apply`,
		planValue.String(), stateValue.String())
}
