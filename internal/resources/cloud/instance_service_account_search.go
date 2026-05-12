package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	cfg := cloudClient.GetConfig()
	basePath, err := cfg.ServerURLWithContext(ctx, "InstancesAPIService.PostInstanceServiceAccounts")
	if err != nil {
		return nil, err
	}

	path, err := url.JoinPath(basePath, "instances", stackSlug, "api", "serviceaccounts", "search")
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("query", query)
	q.Set("page", strconv.FormatInt(page, 10))
	q.Set("perpage", strconv.FormatInt(searchServiceAccountsPerPage, 10))

	u := &url.URL{
		Scheme:   cfg.Scheme,
		Host:     cfg.Host,
		Path:     path,
		RawQuery: q.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range cfg.DefaultHeader {
		req.Header.Add(k, v)
	}
	if cfg.UserAgent != "" {
		req.Header.Add("User-Agent", cfg.UserAgent)
	}

	httpResp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searching instance service accounts for stack %q: HTTP %d: %s", stackSlug, httpResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out models.SearchOrgServiceAccountsResult
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode search service accounts response: %w", err)
	}
	return &out, nil
}
