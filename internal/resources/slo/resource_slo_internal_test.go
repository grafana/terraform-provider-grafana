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

// TestUnit_convertQueryToModel_sourceDatasourceUID covers the read side of
// source_datasource_uid for both query types that carry it.
//
// State mirrors the API verbatim. An SLO created without the field comes back with the
// key absent, which the OpenAPI client decodes as a nil pointer and StringPointerValue
// maps to null. `omitempty` on a *string only skips nil, so an SLO that stores an
// explicit "" round-trips as "" — the API preserves it rather than normalizing it away.
//
// Terraform's own writes never produce "": nonEmptyStringValidator rejects it at plan.
// The "" case is reachable only by adopting an SLO created by another client, where it
// shows up as a single diff that clears on the next apply.
func TestUnit_convertQueryToModel_sourceDatasourceUID(t *testing.T) {
	ptr := func(s string) *string { return &s }

	cases := []struct {
		name      string
		apiValue  *string
		wantNull  bool
		wantValue string
	}{
		{"nil_pointer_simulates_omitted_field", nil, true, ""},
		{"empty_string_preserved_verbatim", ptr(""), false, ""},
		{"populated_uid", ptr("bfrmht0w6t79ca"), false, "bfrmht0w6t79ca"},
	}

	for _, tc := range cases {
		t.Run("freeform/"+tc.name, func(t *testing.T) {
			models, diags := convertQueryToModel(context.Background(), slo.SloV00Query{
				Type: QueryTypeFreeform,
				Freeform: &slo.SloV00FreeformQuery{
					Query:               "sum(rate(up[$__rate_interval]))",
					SourceDatasourceUid: tc.apiValue,
				},
			})
			require.False(t, diags.HasError(), "convertQueryToModel diags: %v", diags)
			require.Len(t, models, 1)
			require.Len(t, models[0].Freeform, 1)

			got := models[0].Freeform[0].SourceDatasourceUID
			require.Equalf(t, tc.wantNull, got.IsNull(), "IsNull mismatch: %s", got.String())
			require.Equal(t, tc.wantValue, got.ValueString())
		})

		t.Run("ratio/"+tc.name, func(t *testing.T) {
			models, diags := convertQueryToModel(context.Background(), slo.SloV00Query{
				Type: QueryTypeRatio,
				Ratio: &slo.SloV00RatioQuery{
					SuccessMetric:       slo.SloV00MetricDef{PrometheusMetric: "success_total"},
					TotalMetric:         slo.SloV00MetricDef{PrometheusMetric: "total"},
					SourceDatasourceUid: tc.apiValue,
				},
			})
			require.False(t, diags.HasError(), "convertQueryToModel diags: %v", diags)
			require.Len(t, models, 1)
			require.Len(t, models[0].Ratio, 1)

			got := models[0].Ratio[0].SourceDatasourceUID
			require.Equalf(t, tc.wantNull, got.IsNull(), "IsNull mismatch: %s", got.String())
			require.Equal(t, tc.wantValue, got.ValueString())
		})
	}
}

// TestUnit_packQuery_sourceDatasourceUID covers the write side: an unset (null or
// unknown) attribute must serialize to an absent field, both so that a create without
// the attribute matches what read returns, and so that removing the attribute from
// HCL actually clears it on the API instead of leaving the previous uid in place.
func TestUnit_packQuery_sourceDatasourceUID(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		configured types.String
		wantNil    bool
		wantValue  string
	}{
		{"null_config_sends_nothing", types.StringNull(), true, ""},
		{"unknown_config_sends_nothing", types.StringUnknown(), true, ""},
		{"populated_config_is_sent", types.StringValue("bfrmht0w6t79ca"), false, "bfrmht0w6t79ca"},
	}

	for _, tc := range cases {
		t.Run("freeform/"+tc.name, func(t *testing.T) {
			apiQuery, diags := packQuery(ctx, queryModel{
				Type: types.StringValue("freeform"),
				Freeform: []freeformQueryModel{{
					Query:               types.StringValue("sum(rate(up[$__rate_interval]))"),
					SourceDatasourceUID: tc.configured,
				}},
			})
			require.False(t, diags.HasError(), "packQuery diags: %v", diags)
			require.NotNil(t, apiQuery.Freeform)

			got := apiQuery.Freeform.SourceDatasourceUid
			if tc.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tc.wantValue, *got)
		})

		t.Run("ratio/"+tc.name, func(t *testing.T) {
			apiQuery, diags := packQuery(ctx, queryModel{
				Type: types.StringValue("ratio"),
				Ratio: []ratioQueryModel{{
					SuccessMetric:       types.StringValue("success_total"),
					TotalMetric:         types.StringValue("total"),
					GroupByLabels:       types.ListNull(types.StringType),
					SourceDatasourceUID: tc.configured,
				}},
			})
			require.False(t, diags.HasError(), "packQuery diags: %v", diags)
			require.NotNil(t, apiQuery.Ratio)

			got := apiQuery.Ratio.SourceDatasourceUid
			if tc.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tc.wantValue, *got)
		})
	}
}

// TestUnit_sourceDatasourceUID_roundTripIsConsistent ties the two halves together
// for the case that actually bites in production: HCL that omits the attribute.
// Plan holds null, pack drops the field, the API returns nothing, and read must
// arrive back at null so plan and state compare equal.
func TestUnit_sourceDatasourceUID_roundTripIsConsistent(t *testing.T) {
	ctx := context.Background()

	planValue := types.StringNull()

	apiQuery, diags := packQuery(ctx, queryModel{
		Type: types.StringValue("freeform"),
		Freeform: []freeformQueryModel{{
			Query:               types.StringValue("sum(rate(up[$__rate_interval]))"),
			SourceDatasourceUID: planValue,
		}},
	})
	require.False(t, diags.HasError())
	require.Nil(t, apiQuery.Freeform.SourceDatasourceUid,
		"omitted attribute must not be serialized at all")

	// Simulate the GET: omitempty drops the absent field, so it decodes as nil.
	models, diags := convertQueryToModel(ctx, slo.SloV00Query{
		Type:     QueryTypeFreeform,
		Freeform: &slo.SloV00FreeformQuery{Query: apiQuery.Freeform.Query},
	})
	require.False(t, diags.HasError())
	stateValue := models[0].Freeform[0].SourceDatasourceUID

	require.True(t, planValue.Equal(stateValue),
		"omitted-attribute round-trip: plan %s != state %s", planValue.String(), stateValue.String())
}

// TestUnit_schemaWiresUpSourceDatasourceUID is the nested-block counterpart to
// TestUnit_schemaWiresUpEmptyStringValidators. source_datasource_uid lives inside
// query.freeform and query.ratio rather than at the top level, so it needs its own
// walk. It also pins the Optional-and-not-Computed decision: the API never
// populates the field on the user's behalf — the documented fallback to
// destination_datasource.uid is resolved when the SLI query runs and is never
// persisted — so marking it Computed would promise a value terraform can never
// read back.
func TestUnit_schemaWiresUpSourceDatasourceUID(t *testing.T) {
	ctx := context.Background()

	var resp resource.SchemaResponse
	(&sloResource{}).Schema(ctx, resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "Schema returned errors: %v", resp.Diagnostics)

	queryBlock, ok := resp.Schema.Blocks["query"].(fwschema.ListNestedBlock)
	require.True(t, ok, "query block is not a ListNestedBlock")

	for _, queryType := range []string{"freeform", "ratio"} {
		t.Run(queryType, func(t *testing.T) {
			inner, ok := queryBlock.NestedObject.Blocks[queryType].(fwschema.ListNestedBlock)
			require.True(t, ok, "query.%s is not a ListNestedBlock", queryType)

			attr, ok := inner.NestedObject.Attributes["source_datasource_uid"]
			require.True(t, ok, "query.%s is missing source_datasource_uid", queryType)

			strAttr, ok := attr.(fwschema.StringAttribute)
			require.True(t, ok, "source_datasource_uid is not a StringAttribute (was %T)", attr)

			require.True(t, strAttr.Optional, "source_datasource_uid must be Optional")
			require.False(t, strAttr.Computed,
				"source_datasource_uid must not be Computed — the API never defaults it, "+
					"so a Computed attribute would stay unknown and never resolve")

			require.NotEmpty(t, strAttr.Validators,
				"source_datasource_uid has no String validators wired up — an empty config "+
					"value would pack to nil, read back as null, and trigger the post-apply "+
					"inconsistent-result error")

			rejected := false
			joined := ""
			for _, v := range strAttr.Validators {
				vresp := validator.StringResponse{}
				v.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue("")}, &vresp)
				if vresp.Diagnostics.HasError() {
					rejected = true
					for _, d := range vresp.Diagnostics {
						joined += d.Detail() + "\n"
					}
				}
			}
			require.True(t, rejected, "no validator rejected an empty source_datasource_uid")
			require.Contains(t, joined, "omit the attribute entirely to run the query against the destination datasource",
				"validator error did not tell the user how to fix the empty value; got %q", joined)
		})
	}
}
