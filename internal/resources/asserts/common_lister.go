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

		// Get stack ID from lister data
		stackID, ok := data.(string)
		if !ok || stackID == "" {
			return nil, fmt.Errorf("stack ID is required for listing Asserts resources")
		}

		return listerFunc(ctx, client.AssertsAPIClient, stackID)
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
			// Use the format "stackID:name" since that's our resource ID format
			names = append(names, fmt.Sprintf("%s:%s", stackID, *config.Name))
		}
	}
	return names, nil
}

// listDisabledAlertConfigs retrieves the list of all disabled alert configuration names for a specific stack
func listDisabledAlertConfigs(ctx context.Context, client *assertsapi.APIClient, stackID string) ([]string, error) {
	request := client.DisabledAlertConfigControllerAPI.GetAllDisabledAlertConfigs(ctx).
		XScopeOrgID(stackID)

	configs, _, err := request.Execute()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, config := range configs.DisabledAlertConfigs {
		if config.Name != nil {
			// Use the format "stackID:name" since that's our resource ID format
			names = append(names, fmt.Sprintf("%s:%s", stackID, *config.Name))
		}
	}
	return names, nil
}
