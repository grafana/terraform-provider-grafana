package asserts

import (
	"context"
	"fmt"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// assertsListerFunction is a helper function that wraps a lister function to be used more easily in Asserts resources.
func assertsListerFunction(listerFunc func(ctx context.Context, client *assertsapi.APIClient, stackID string) ([]string, error)) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *common.Client, data any) ([]string, error) {
		if client.AssertsAPIClient == nil {
			return nil, fmt.Errorf("client not configured for the Asserts API")
		}

		// Get stack ID from provider configuration
		stackID := client.GrafanaStackID
		if stackID == 0 {
			return nil, fmt.Errorf("stack_id must be set in provider configuration for Asserts resources")
		}

		return listerFunc(ctx, client.AssertsAPIClient, fmt.Sprintf("%d", stackID))
	}
}

// listAlertConfigs retrieves the list of all alert configuration names for a specific stack
func listAlertConfigs(ctx context.Context, client *assertsapi.APIClient, stackID string) ([]string, error) {
	request := client.AlertConfigurationAPI.GetAllAlertConfigs(ctx).
		XScopeOrgID(stackID)

	alertConfigs, _, err := request.Execute()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, config := range alertConfigs.AlertConfigs {
		if config.Name != nil {
			// Resource ID is just the name now (stack ID from provider config)
			names = append(names, *config.Name)
		}
	}
	return names, nil
}

// listDisabledAlertConfigs retrieves the list of all disabled alert configuration names for a specific stack
func listDisabledAlertConfigs(ctx context.Context, client *assertsapi.APIClient, stackID string) ([]string, error) {
	request := client.AlertConfigurationAPI.GetAllDisabledAlertConfigs(ctx).
		XScopeOrgID(stackID)

	configs, _, err := request.Execute()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, config := range configs.DisabledAlertConfigs {
		if config.Name != nil {
			// Resource ID is just the name now (stack ID from provider config)
			names = append(names, *config.Name)
		}
	}
	return names, nil
}
