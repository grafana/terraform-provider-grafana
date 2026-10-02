package agento11y

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
)

func TestUnitEvaluationRuleSchemaIncludesExecutionAndTagFields(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	(&evaluationRuleResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	executionMode, ok := resp.Schema.Attributes["execution_mode"].(schema.StringAttribute)
	if !ok {
		t.Fatal("execution_mode is not a string attribute")
	}
	if !executionMode.Optional || !executionMode.Computed {
		t.Fatal("execution_mode must be optional and computed")
	}

	filterableTagKeys, ok := resp.Schema.Attributes["filterable_tag_keys"].(schema.ListAttribute)
	if !ok {
		t.Fatal("filterable_tag_keys is not a list attribute")
	}
	if !filterableTagKeys.Optional || filterableTagKeys.ElementType != types.StringType {
		t.Fatal("filterable_tag_keys must be an optional list of strings")
	}
}

func TestUnitEvaluationRuleToModelIncludesExecutionAndTagFields(t *testing.T) {
	t.Parallel()

	model, diags := ruleToModel(context.Background(), agento11yapi.Rule{
		RuleID:            "quality-gate",
		ExecutionMode:     executionModeSequential,
		FilterableTagKeys: []string{"environment", "team"},
	})
	if diags.HasError() {
		t.Fatalf("ruleToModel diagnostics: %v", diags)
	}
	if model.ExecutionMode != types.StringValue(executionModeSequential) {
		t.Fatalf("execution_mode = %v, want %q", model.ExecutionMode, executionModeSequential)
	}

	var gotTagKeys []string
	diags = model.FilterableTagKeys.ElementsAs(context.Background(), &gotTagKeys, false)
	if diags.HasError() {
		t.Fatalf("filterable_tag_keys diagnostics: %v", diags)
	}
	if len(gotTagKeys) != 2 || gotTagKeys[0] != "environment" || gotTagKeys[1] != "team" {
		t.Fatalf("filterable_tag_keys = %v, want [environment team]", gotTagKeys)
	}
}

func TestUnitEvaluationRuleRejectsConversationFilterableTags(t *testing.T) {
	t.Parallel()

	tagKeys, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"team"})
	if diags.HasError() {
		t.Fatalf("build filterable_tag_keys: %v", diags)
	}
	got := validateRuleSelection(selectorConversation, types.Int64Value(60), tagKeys)
	if !got.HasError() {
		t.Fatal("expected conversation rule with filterable_tag_keys to be rejected")
	}
}
