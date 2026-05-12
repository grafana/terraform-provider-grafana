package cloud

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/grafana-openapi-client-go/models"
)

const searchServiceAccountsPerPage = int64(200)

// findStackServiceAccountByExactName returns a service account on the stack whose display name equals name.
// It uses the Grafana instance search API via the Cloud API proxy. Returns (nil, nil) when not found.
func findStackServiceAccountByExactName(ctx context.Context, cloudClient *gcom.APIClient, stackSlug, name string) (*models.ServiceAccountDTO, error) {
	var page int64 = 1
	for {
		result, err := searchStackInstanceServiceAccountsPage(ctx, cloudClient, stackSlug, name, page)
		if err != nil {
			return nil, err
		}
		for _, sa := range result.ServiceAccounts {
			if sa != nil && sa.Name == name {
				return sa, nil
			}
		}
		perPage := result.PerPage
		if perPage == 0 {
			perPage = int64(len(result.ServiceAccounts))
		}
		if perPage == 0 {
			break
		}
		if page*perPage >= result.TotalCount || int64(len(result.ServiceAccounts)) == 0 {
			break
		}
		page++
	}
	return nil, nil
}

func searchStackInstanceServiceAccountsPage(ctx context.Context, cloudClient *gcom.APIClient, stackSlug, query string, page int64) (*models.SearchOrgServiceAccountsResult, error) {
	gResp, httpResp, err := cloudClient.InstancesAPI.GetInstanceServiceAccountsSearch(ctx, stackSlug).
		Query(query).
		// nolint: gosec
		Page(int32(page)).
		// nolint: gosec
		Perpage(int32(searchServiceAccountsPerPage)).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("search instance service accounts for stack %q: %w", stackSlug, err)
	}
	if httpResp != nil && httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searching instance service accounts for stack %q: unexpected HTTP %d", stackSlug, httpResp.StatusCode)
	}
	if gResp == nil {
		return nil, fmt.Errorf("search instance service accounts for stack %q: empty response", stackSlug)
	}

	out := &models.SearchOrgServiceAccountsResult{
		TotalCount: int64(gResp.GetTotalCount()),
		Page:       int64(gResp.GetPage()),
		PerPage:    int64(gResp.GetPerPage()),
	}
	for _, inner := range gResp.GetServiceAccounts() {
		out.ServiceAccounts = append(out.ServiceAccounts, &models.ServiceAccountDTO{
			ID:   int64(inner.GetId()),
			Name: inner.GetName(),
		})
	}
	return out, nil
}
