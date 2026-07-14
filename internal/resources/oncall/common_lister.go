package oncall

import (
	"context"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

type listerFunc func(client *onCallAPI.Client, listOptions onCallAPI.ListOptions) (ids []string, nextPage *string, err error)

// oncallListerFunction is a helper function that wraps a lister function be used more easily in oncall resources.
func oncallListerFunction(listerFunc listerFunc) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *common.Client, data any) ([]string, error) {
		if client.OnCallClient == nil {
			return nil, fmt.Errorf("client not configured for Grafana OnCall API")
		}
		ids := []string{}
		page := 1
		for {
			newIDs, nextPage, err := listerFunc(client.OnCallClient, onCallAPI.ListOptions{Page: page})
			if err != nil {
				return nil, err
			}
			ids = append(ids, newIDs...)
			if nextPage == nil {
				break
			}
			page++
		}
		return ids, nil
	}
}
