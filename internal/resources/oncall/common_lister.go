package oncall

import (
	"context"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// oncallListerFunction is a helper function that wraps a lister function be used more easily in oncall resources.
func oncallListerFunction(listerFunc func(ctx context.Context, client *onCallAPI.Client) ([]string, error)) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *common.Client, data any) ([]string, error) {
		if client.OnCallClient == nil {
			return nil, fmt.Errorf("client not configured for Grafana OnCall API")
		}
		return listerFunc(ctx, client.OnCallClient)
	}
}
