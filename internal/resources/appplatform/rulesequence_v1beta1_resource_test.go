package appplatform

import (
	"context"
	"testing"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestParseRuleSequenceV1Beta1Spec(t *testing.T) {
	var dst v1beta1.RuleSequence
	diags := parseRuleSequenceV1Beta1Spec(context.Background(), ruleSequenceV1Beta1SpecObject(t, []string{"alert"}), &dst)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, v1beta1.RuleSequenceSpec{
		Trigger:        v1beta1.RuleSequenceIntervalTrigger{Interval: "1m"},
		RecordingRules: []v1beta1.RuleSequenceRuleRef{{Name: "recording"}},
		AlertRules:     []v1beta1.RuleSequenceRuleRef{{Name: "alert"}},
	}, dst.Spec)
}

func TestSaveRuleSequenceV1Beta1SpecRoundTrip(t *testing.T) {
	for name, alertRules := range map[string][]string{
		"with alert rules":    {"alert-1", "alert-2"},
		"without alert rules": nil,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			src := ruleSequenceV1Beta1SpecObject(t, alertRules)

			var seq v1beta1.RuleSequence
			diags := parseRuleSequenceV1Beta1Spec(ctx, src, &seq)
			require.False(t, diags.HasError(), diags)

			var dst ResourceModel
			diags = saveRuleSequenceV1Beta1Spec(ctx, &seq, &dst)
			require.False(t, diags.HasError(), diags)
			require.True(t, dst.Spec.Type(ctx).Equal(ruleSequenceSpecV1Beta1Type))
			require.True(t, dst.Spec.Equal(src), "got %s, want %s", dst.Spec, src)
		})
	}
}

func ruleSequenceV1Beta1SpecObject(t *testing.T, alertRules []string) types.Object {
	t.Helper()
	ctx := context.Background()

	trigger, diags := types.ObjectValueFrom(ctx, ruleTriggerType.AttrTypes, RuleTriggerModel{Interval: types.StringValue("1m")})
	require.False(t, diags.HasError(), diags)

	refs := func(names []string) types.List {
		if names == nil {
			return types.ListNull(ruleSequenceRuleRefType)
		}
		models := make([]ruleSequenceRuleRefModel, 0, len(names))
		for _, name := range names {
			models = append(models, ruleSequenceRuleRefModel{Name: types.StringValue(name)})
		}
		list, diags := types.ListValueFrom(ctx, ruleSequenceRuleRefType, models)
		require.False(t, diags.HasError(), diags)
		return list
	}

	spec, diags := types.ObjectValue(ruleSequenceSpecV1Beta1Type.AttrTypes, map[string]attr.Value{
		"trigger":         trigger,
		"recording_rules": refs([]string{"recording"}),
		"alert_rules":     refs(alertRules),
	})
	require.False(t, diags.HasError(), diags)
	return spec
}
