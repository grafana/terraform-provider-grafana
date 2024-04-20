package grafana

import (
	"context"
	"fmt"
	"sync"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/orgs"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
)

// ListerData is used as the data arg in "ListIDs" functions. It allows getting data common to multiple resources.
type ListerData struct {
	singleOrg bool
	orgIDs    []int64
	orgsInit  sync.Once
}

func NewListerData(singleOrg bool) *ListerData {
	return &ListerData{
		singleOrg: singleOrg,
	}
}

func (ld *ListerData) OrgIDs(client *goapi.GrafanaHTTPAPI) ([]int64, error) {
	if ld.singleOrg {
		return nil, nil
	}

	var err error
	ld.orgsInit.Do(func() {
		client = client.Clone().WithOrgID(0)

		var page int64 = 1
		for {
			var resp *orgs.SearchOrgsOK
			if resp, err = client.Orgs.SearchOrgs(orgs.NewSearchOrgsParams().WithPage(&page)); err != nil {
				return
			}
			for _, org := range resp.Payload {
				ld.orgIDs = append(ld.orgIDs, org.ID)
			}
			if len(resp.Payload) == 0 {
				break
			}
			page++
		}
	})
	if err != nil {
		return nil, err
	}

	return ld.orgIDs, nil
}

// listerFunction is a helper function that wraps a lister function be used more easily in grafana resources.
func listerFunction(listerFunc func(ctx context.Context, client *goapi.GrafanaHTTPAPI, data *ListerData) ([]string, error)) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *common.Client, data any) ([]string, error) {
		lm, ok := data.(*ListerData)
		if !ok {
			return nil, fmt.Errorf("unexpected data type: %T", data)
		}
		if client.GrafanaAPI == nil {
			return nil, fmt.Errorf("client not configured for Grafana API")
		}
		return listerFunc(ctx, client.GrafanaAPI, lm)
	}
}
