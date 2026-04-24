package fleetmanagement

import (
	"context"
	"testing"

	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/require"
)

const testPipelineAlloyContents = "logging {}"

func TestPipelineMessageToModel(t *testing.T) {
	name := "test_name"
	contents := testPipelineAlloyContents
	matcher1 := "collector.os=\"linux\""
	matcher2 := "owner=\"TEAM-A\""
	enabled := true
	id := "123"

	msg := &pipelinev1.Pipeline{
		Name:       name,
		Contents:   contents,
		Matchers:   []string{matcher1, matcher2},
		Enabled:    &enabled,
		Id:         &id,
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
	}

	expectedModel := &pipelineModel{
		Name:     types.StringValue(name),
		Contents: NewPipelineConfigValue(contents),
		Matchers: NewListOfPrometheusMatcherValueMust(
			[]attr.Value{
				basetypes.NewStringValue(matcher1),
				basetypes.NewStringValue(matcher2),
			},
		),
		Enabled:                  types.BoolPointerValue(&enabled),
		ID:                       types.StringPointerValue(&id),
		ConfigType:               types.StringValue("ALLOY"),
		TerraformSourceNamespace: types.StringValue(defaultTerraformPipelineSourceNamespace),
	}

	ctx := context.Background()
	actualModel, diags := pipelineMessageToModel(ctx, msg, nil)
	require.False(t, diags.HasError())
	require.Equal(t, expectedModel, actualModel)
}

func TestPipelineModelToMessage(t *testing.T) {
	name := "test_name"
	contents := testPipelineAlloyContents
	matcher1 := "collector.os=\"linux\""
	matcher2 := "owner=\"TEAM-A\""
	enabled := true
	id := "123"

	model := &pipelineModel{
		Name:     types.StringValue(name),
		Contents: NewPipelineConfigValue(contents),
		Matchers: NewListOfPrometheusMatcherValueMust(
			[]attr.Value{
				basetypes.NewStringValue(matcher1),
				basetypes.NewStringValue(matcher2),
			},
		),
		Enabled:                  types.BoolPointerValue(&enabled),
		ID:                       types.StringPointerValue(&id),
		ConfigType:               types.StringValue("ALLOY"),
		TerraformSourceNamespace: types.StringValue(defaultTerraformPipelineSourceNamespace),
	}

	expectedMsg := &pipelinev1.Pipeline{
		Name:       name,
		Contents:   contents,
		Matchers:   []string{matcher1, matcher2},
		Enabled:    &enabled,
		Id:         &id,
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
		Source: &pipelinev1.PipelineSource{
			Type:      pipelinev1.PipelineSource_SOURCE_TYPE_TERRAFORM,
			Namespace: defaultTerraformPipelineSourceNamespace,
		},
	}

	ctx := context.Background()
	actualMsg, diags := pipelineModelToMessage(ctx, model)
	require.False(t, diags.HasError())
	require.Equal(t, expectedMsg, actualMsg)
}

func TestStringSliceToMatcherValues(t *testing.T) {
	tests := []struct {
		name        string
		nativeSlice []string
		expected    ListOfPrometheusMatcherValue
	}{
		{
			"nil slice",
			nil,
			NewListOfPrometheusMatcherValueMust([]attr.Value{}),
		},
		{
			"empty slice",
			[]string{},
			NewListOfPrometheusMatcherValueMust([]attr.Value{}),
		},
		{
			"non-empty slice",
			[]string{
				"collector.os=linux",
				"collector.os=darwin",
			},
			NewListOfPrometheusMatcherValueMust(
				[]attr.Value{
					basetypes.NewStringValue("collector.os=linux"),
					basetypes.NewStringValue("collector.os=darwin"),
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, diags := stringSliceToMatcherValues(ctx, tt.nativeSlice)
			require.False(t, diags.HasError())
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestMatcherValuesToStringSlice(t *testing.T) {
	tests := []struct {
		name        string
		genericList ListOfPrometheusMatcherValue
		expected    []string
	}{
		{
			"null list",
			NewListOfPrometheusMatcherValueNull(),
			[]string{},
		},
		{
			"unknown list",
			NewListOfPrometheusMatcherValueUnknown(),
			[]string{},
		},
		{
			"empty list",
			NewListOfPrometheusMatcherValueMust([]attr.Value{}),
			[]string{},
		},
		{
			"non-empty list",
			NewListOfPrometheusMatcherValueMust(
				[]attr.Value{
					basetypes.NewStringValue("collector.os=linux"),
					basetypes.NewStringValue("collector.os=darwin"),
				},
			),
			[]string{
				"collector.os=linux",
				"collector.os=darwin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, diags := matcherValuesToStringSlice(ctx, tt.genericList)
			require.False(t, diags.HasError())
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestPipelineMessageToModel_WithTerraformSourceNamespace(t *testing.T) {
	ns := "my-workspace"
	msg := &pipelinev1.Pipeline{
		Name:       "p",
		Contents:   testPipelineAlloyContents,
		Matchers:   []string{},
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
		Source: &pipelinev1.PipelineSource{
			Type:      pipelinev1.PipelineSource_SOURCE_TYPE_TERRAFORM,
			Namespace: ns,
		},
	}

	ctx := context.Background()
	m, diags := pipelineMessageToModel(ctx, msg, nil)
	require.False(t, diags.HasError())
	require.Equal(t, ns, m.TerraformSourceNamespace.ValueString())
}

func TestPipelineModelToMessage_CustomTerraformSourceNamespace(t *testing.T) {
	model := &pipelineModel{
		Name:                     types.StringValue("p"),
		Contents:                 NewPipelineConfigValue(testPipelineAlloyContents),
		Matchers:                 NewListOfPrometheusMatcherValueMust([]attr.Value{}),
		Enabled:                  types.BoolValue(true),
		ID:                       types.StringValue("id-1"),
		ConfigType:               types.StringValue("ALLOY"),
		TerraformSourceNamespace: types.StringValue("prod/root"),
	}

	ctx := context.Background()
	msg, diags := pipelineModelToMessage(ctx, model)
	require.False(t, diags.HasError())
	require.NotNil(t, msg.Source)
	require.Equal(t, pipelinev1.PipelineSource_SOURCE_TYPE_TERRAFORM, msg.Source.Type)
	require.Equal(t, "prod/root", msg.Source.Namespace)
}

// https://github.com/grafana/terraform-provider-grafana/issues/2632
func TestPipelineMessageToModel_PrefersPlannedContentsWhenSemanticallyEqual(t *testing.T) {
	planned := testPipelineAlloyContents
	apiFormatted := testPipelineAlloyContents + "\n"

	eq, err := alloyConfigEqual(planned, apiFormatted)
	require.NoError(t, err)
	require.True(t, eq)

	msg := &pipelinev1.Pipeline{
		Name:       "p",
		Contents:   apiFormatted,
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
	}
	prefs := &pipelineModel{Contents: NewPipelineConfigValue(planned)}

	ctx := context.Background()
	model, diags := pipelineMessageToModel(ctx, msg, prefs)
	require.False(t, diags.HasError())
	require.Equal(t, planned, model.Contents.ValueString())
}

func TestPipelineMessageToModel_FillsOmittedEnabledFromPlan(t *testing.T) {
	msg := &pipelinev1.Pipeline{
		Name:       "p",
		Contents:   testPipelineAlloyContents,
		Matchers:   []string{},
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
		Enabled:    nil,
	}
	prefs := &pipelineModel{
		Contents: NewPipelineConfigValue(testPipelineAlloyContents),
		Enabled:  types.BoolValue(false),
	}

	ctx := context.Background()
	m, diags := pipelineMessageToModel(ctx, msg, prefs)
	require.False(t, diags.HasError())
	require.False(t, m.Enabled.ValueBool())

	m2, diags := pipelineMessageToModel(ctx, msg, &pipelineModel{
		Contents: NewPipelineConfigValue(testPipelineAlloyContents),
		Enabled:  types.BoolValue(true),
	})
	require.False(t, diags.HasError())
	require.True(t, m2.Enabled.ValueBool())
}

func TestPipelineMessageToModel_DefaultsEnabledTrueWhenOmittedAndNoPlan(t *testing.T) {
	msg := &pipelinev1.Pipeline{
		Name:       "p",
		Contents:   testPipelineAlloyContents,
		Matchers:   []string{},
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
		Enabled:    nil,
	}
	ctx := context.Background()
	m, diags := pipelineMessageToModel(ctx, msg, nil)
	require.False(t, diags.HasError())
	require.True(t, m.Enabled.ValueBool())
}

func TestPipelineMessageToModel_FillsOmittedConfigTypeFromPlan(t *testing.T) {
	msg := &pipelinev1.Pipeline{
		Name:       "p",
		Contents:   testPipelineAlloyContents,
		Matchers:   []string{},
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_UNSPECIFIED,
	}
	prefs := &pipelineModel{
		Contents:   NewPipelineConfigValue(testPipelineAlloyContents),
		ConfigType: types.StringValue(ConfigTypeOtel),
	}

	ctx := context.Background()
	m, diags := pipelineMessageToModel(ctx, msg, prefs)
	require.False(t, diags.HasError())
	require.Equal(t, ConfigTypeOtel, m.ConfigType.ValueString())
}

func TestPipelineMessageToModel_DefaultsConfigTypeAlloyWhenOmittedAndNoPlan(t *testing.T) {
	msg := &pipelinev1.Pipeline{
		Name:       "p",
		Contents:   testPipelineAlloyContents,
		Matchers:   []string{},
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_UNSPECIFIED,
	}
	ctx := context.Background()
	m, diags := pipelineMessageToModel(ctx, msg, nil)
	require.False(t, diags.HasError())
	require.Equal(t, ConfigTypeAlloy, m.ConfigType.ValueString())
}

func TestReconcilePipelineModelForApply_TypicalFixtureSecondPassNoOp(t *testing.T) {
	enabled := true
	id := "pipe-1"
	msg := &pipelinev1.Pipeline{
		Name:       "p",
		Contents:   testPipelineAlloyContents,
		Matchers:   []string{},
		Enabled:    &enabled,
		Id:         &id,
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
	}
	prefs := &pipelineModel{
		Contents:                 NewPipelineConfigValue(testPipelineAlloyContents),
		Enabled:                  types.BoolValue(true),
		ConfigType:               types.StringValue(ConfigTypeAlloy),
		TerraformSourceNamespace: types.StringNull(),
	}

	ctx := context.Background()
	first, diags := pipelineMessageToModel(ctx, msg, prefs)
	require.False(t, diags.HasError())
	out, diags := reconcilePipelineModelForApply(ctx, msg, prefs)
	require.False(t, diags.HasError())
	second, diags2 := pipelineMessageToModel(ctx, msg, first)
	require.False(t, diags2.HasError())
	require.Equal(t, second, out, "reconcile should equal a manual second pass with first as prefs")
}
