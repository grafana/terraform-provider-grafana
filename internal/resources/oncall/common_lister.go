package oncall

import (
	"context"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// ListerData is used as the data arg in "ListIDs" functions. It allows getting data common to multiple resources.
type ListerData struct {
}

func NewListerData() *ListerData {
	return &ListerData{}
}

// oncallListerFunction is a helper function that wraps a lister function be used more easily in oncall resources.
func oncallListerFunction(listerFunc func(ctx context.Context, client *onCallAPI.Client, listerData *ListerData) ([]string, error)) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *common.Client, data any) ([]string, error) {
		ld, ok := data.(*ListerData)
		if !ok {
			return nil, fmt.Errorf("unexpected data type: %T", data)
		}
		if client.GrafanaCloudAPI == nil {
			return nil, fmt.Errorf("client not configured for Grafana Cloud API")
		}
		return listerFunc(ctx, client.OnCallClient, ld)
	}
}
