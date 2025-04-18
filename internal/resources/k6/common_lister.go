package k6

import (
	"context"
	"fmt"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/k6providerapi"
)

// k6ListerFunction is a helper function that wraps a lister function be used more easily in k6 resources.
func k6ListerFunction(listerFunc func(ctx context.Context, client *k6.APIClient, config *k6providerapi.K6APIConfig) ([]string, error)) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *common.Client, data any) ([]string, error) {
		if client.K6APIClient == nil || client.K6APIConfig == nil {
			return nil, fmt.Errorf("client not configured for the k6 Cloud API")
		}
		return listerFunc(ctx, client.K6APIClient, client.K6APIConfig)
	}
}
