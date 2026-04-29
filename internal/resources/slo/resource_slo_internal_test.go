package slo

import (
	"context"
	"testing"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

// TestUnit_schemaWiresUpEmptyStringValidators is the regression guard for the
// schema → validator wiring on folder_uid and search_expression.
// TestUnit_nonEmptyStringValidator below proves the validator *type* behaves
// correctly in isolation, but a future refactor could drop the
// `Validators: []validator.String{...}` block off either attribute and that
// suite would still pass. This test loads the actual resource schema, walks
// to the attribute, runs every wired-up String validator with `""`, and
// asserts the error fires with a substring of the user-facing remediation —
// so a change of fieldName, message, or wholesale validator removal all
// fail loudly.
func TestUnit_schemaWiresUpEmptyStringValidators(t *testing.T) {
	ctx := context.Background()

	var resp resource.SchemaResponse
	(&sloResource{}).Schema(ctx, resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "Schema returned errors: %v", resp.Diagnostics)

	cases := []struct {
		attrName   string
		wantSubstr string // substring of the expected error detail; covers fieldName + remediation hint
	}{
		{"folder_uid", "associate the SLO with the default Grafana SLO folder"},
		{"search_expression", "omit the attribute entirely to leave it unset"},
	}

	for _, tc := range cases {
		t.Run(tc.attrName, func(t *testing.T) {
			attr, ok := resp.Schema.Attributes[tc.attrName]
			require.True(t, ok, "schema is missing attribute %q", tc.attrName)

			strAttr, ok := attr.(fwschema.StringAttribute)
			require.True(t, ok, "attribute %q is not a StringAttribute (was %T)", tc.attrName, attr)

			validators := strAttr.Validators
			require.NotEmpty(t, validators,
				"attribute %q has no String validators wired up — "+
					"empty config values would silently round-trip and trigger the post-apply "+
					"inconsistent-result error in production", tc.attrName)

			// Run every wired validator with `""` and require at least one to
			// reject with a remediation hint. This catches both removal of
			// the validator and silent replacement with one that doesn't tell
			// the user how to fix the input.
			req := validator.StringRequest{ConfigValue: types.StringValue("")}
			var allDiags []string
			rejected := false
			for _, v := range validators {
				vresp := validator.StringResponse{}
				v.ValidateString(ctx, req, &vresp)
				if vresp.Diagnostics.HasError() {
					rejected = true
					for _, d := range vresp.Diagnostics {
						allDiags = append(allDiags, d.Detail())
					}
				}
			}

			require.True(t, rejected,
				"no String validator on %q rejected an empty config value", tc.attrName)
			joined := ""
			for _, d := range allDiags {
				joined += d + "\n"
			}
			require.Contains(t, joined, tc.wantSubstr,
				"%q validator error did not mention how to fix the empty value; got %v",
				tc.attrName, allDiags)
		})
	}
}

// TestUnit_nonEmptyStringValidator covers the validator behind folder_uid and
// search_expression. Both attributes hit "Provider produced inconsistent
// result after apply" prior to this validator landing — the user's HCL
// `foo = ""` packed silently to nil at the wire layer, the API stored
// nothing, and read returned null. Catching empty config strings up front
// surfaces a clear actionable error before any API call.
//
// The validator's `message` field overrides the default detail so error text
// can tell the user exactly how to fix it (e.g. "omit the attribute entirely
// to associate the SLO with the default Grafana SLO folder").
func TestUnit_nonEmptyStringValidator(t *testing.T) {
	cases := []struct {
		name             string
		validator        nonEmptyStringValidator
		input            types.String
		wantError        bool
		wantDetailSubstr string
	}{
		{
			"empty_string_default_message",
			nonEmptyStringValidator{fieldName: "uid"},
			types.StringValue(""),
			true,
			"uid must be a non-empty string",
		},
		{
			"empty_string_custom_message_folder_uid",
			nonEmptyStringValidator{
				fieldName: "folder_uid",
				message:   "folder_uid must be non-empty if set; omit the attribute entirely to associate the SLO with the default Grafana SLO folder",
			},
			types.StringValue(""),
			true,
			"omit the attribute entirely to associate the SLO with the default Grafana SLO folder",
		},
		{
			"empty_string_custom_message_search_expression",
			nonEmptyStringValidator{
				fieldName: "search_expression",
				message:   "search_expression must be non-empty if set; omit the attribute entirely to leave it unset",
			},
			types.StringValue(""),
			true,
			"omit the attribute entirely to leave it unset",
		},
		{
			"populated_string_passes",
			nonEmptyStringValidator{fieldName: "folder_uid"},
			types.StringValue("some-uid"),
			false,
			"",
		},
		{
			"null_passes_through",
			nonEmptyStringValidator{fieldName: "folder_uid"},
			types.StringNull(),
			false,
			"",
		},
		{
			"unknown_passes_through",
			nonEmptyStringValidator{fieldName: "folder_uid"},
			types.StringUnknown(),
			false,
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{ConfigValue: tc.input}
			resp := validator.StringResponse{}
			tc.validator.ValidateString(context.Background(), req, &resp)

			if tc.wantError {
				require.True(t, resp.Diagnostics.HasError(),
					"expected validator to error for input %s", tc.input.String())
				detail := resp.Diagnostics[0].Detail()
				require.Contains(t, detail, tc.wantDetailSubstr,
					"error detail %q should contain %q", detail, tc.wantDetailSubstr)
			} else {
				require.False(t, resp.Diagnostics.HasError(),
					"expected validator to pass, got: %v", resp.Diagnostics)
			}
		})
	}
}
