package agento11y

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// listCollectionIDs lists the IDs of all collections visible to the caller.
func listCollectionIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.Agento11yAPIClient == nil {
		return nil, nil
	}
	collections, err := client.Agento11yAPIClient.ListCollections(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(collections))
	for _, c := range collections {
		ids = append(ids, c.CollectionID)
	}
	return ids, nil
}

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
