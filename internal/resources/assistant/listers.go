package assistant

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// listRuleIDs lists the IDs of all assistant rules visible to the caller.
func listRuleIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.AssistantAPIClient == nil {
		return nil, nil
	}
	rules, err := client.AssistantAPIClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// listSkillIDs lists the IDs of all assistant skills visible to the caller.
func listSkillIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.AssistantAPIClient == nil {
		return nil, nil
	}
	skills, err := client.AssistantAPIClient.ListSkills(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(skills))
	for _, s := range skills {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

// listQuickstartIDs lists the IDs of all assistant quickstarts visible to the caller.
func listQuickstartIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.AssistantAPIClient == nil {
		return nil, nil
	}
	quickstarts, err := client.AssistantAPIClient.ListQuickstarts(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(quickstarts))
	for _, q := range quickstarts {
		ids = append(ids, q.ID)
	}
	return ids, nil
}

// listMCPServerIDs lists the IDs of all assistant MCP server integrations visible to the caller.
func listMCPServerIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.AssistantAPIClient == nil {
		return nil, nil
	}
	integrations, err := client.AssistantAPIClient.ListIntegrations(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(integrations))
	for _, i := range integrations {
		ids = append(ids, i.ID)
	}
	return ids, nil
}
