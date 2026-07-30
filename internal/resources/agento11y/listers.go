package agento11y

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// listEvaluatorIDs lists the IDs of all evaluators visible to the caller.
func listEvaluatorIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.Agento11yAPIClient == nil {
		return nil, nil
	}
	evaluators, err := client.Agento11yAPIClient.ListEvaluators(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(evaluators))
	for _, e := range evaluators {
		ids = append(ids, e.EvaluatorID)
	}
	return ids, nil
}

// listRuleIDs lists the IDs of all evaluation rules visible to the caller.
func listRuleIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.Agento11yAPIClient == nil {
		return nil, nil
	}
	rules, err := client.Agento11yAPIClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.RuleID)
	}
	return ids, nil
}

// listHookRuleIDs lists the IDs of all hook rules visible to the caller.
func listHookRuleIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.Agento11yAPIClient == nil {
		return nil, nil
	}
	rules, err := client.Agento11yAPIClient.ListHookRules(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.RuleID)
	}
	return ids, nil
}
