package grafana

import (
	"context"
	"fmt"
	"strings"
	"sync"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/orgs"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
)

// ListerData is used as the data arg in "ListIDs" functions. It allows getting data common to multiple resources.
type ListerData struct {
	omitSingleOrgID bool
	singleOrg       bool
	orgIDs          []int64
	orgsInit        sync.Once
}

func NewListerData(singleOrg, omitSingleOrgID bool) *ListerData {
	return &ListerData{
		singleOrg:       singleOrg,
		omitSingleOrgID: omitSingleOrgID,
	}
}

func (ld *ListerData) OrgIDs(client *goapi.GrafanaHTTPAPI) ([]int64, error) {
	if ld.singleOrg {
		return []int64{0}, nil
	}

	var err error
	ld.orgsInit.Do(func() {
		client = client.Clone().WithOrgID(0)

		var page int64 = 0
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

type grafanaListerFunc func(ctx context.Context, client *goapi.GrafanaHTTPAPI, data *ListerData) ([]string, error)
type grafanaOrgResourceListerFunc func(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error)

// listerFunction is a helper function that wraps a lister function be used more easily in grafana resources.
func listerFunction(listerFunc grafanaListerFunc) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *client.Client, data any) ([]string, error) {
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

func listerFunctionOrgResource(listerFunc grafanaOrgResourceListerFunc) common.ResourceListIDsFunc {
	return listerFunction(func(ctx context.Context, client *goapi.GrafanaHTTPAPI, data *ListerData) ([]string, error) {
		orgIDs, err := data.OrgIDs(client)
		if err != nil {
			return nil, err
		}

		var ids []string
		for _, orgID := range orgIDs {
			idsInOrg, err := listerFunc(ctx, client.Clone().WithOrgID(orgID), orgID)
			if err != nil {
				return nil, err
			}

			// Trim org ID from IDs if there is only one org and it's the default org
			if len(orgIDs) == 1 && (orgID <= 1) && data.omitSingleOrgID {
				for _, id := range idsInOrg {
					ids = append(ids, strings.TrimPrefix(id, fmt.Sprintf("%d:", orgID)))
				}
			} else {
				ids = append(ids, idsInOrg...)
			}
		}

		return ids, nil
	})
}
