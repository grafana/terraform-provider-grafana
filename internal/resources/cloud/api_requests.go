package cloud

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// This file centralises the grafana.com read calls that are made from more than one place so
// they all share the same retry policy (via common.RetryRequest) and the same operation label.
// Keeping a single implementation per call avoids the retry behaviour and log messages drifting
// apart between resources, data sources and listers.

// getOrgWithRetry fetches a Grafana Cloud organization by numeric ID or slug, retrying
// transient grafana.com errors.
func getOrgWithRetry(ctx context.Context, client *gcom.APIClient, idOrSlug string) (*gcom.FormattedApiOrgPublic, error) {
	var org *gcom.FormattedApiOrgPublic
	err := common.RetryRequest(ctx, "get cloud organization", func() (*http.Response, error) {
		o, httpResp, err := client.OrgsAPI.GetOrg(ctx, idOrSlug).Execute()
		org = o
		return httpResp, err
	})
	return org, err
}

// listStackRegionsWithRetry lists all Grafana Cloud stack regions, retrying transient
// grafana.com errors.
func listStackRegionsWithRetry(ctx context.Context, client *gcom.APIClient) (*gcom.GetStackRegions200Response, error) {
	var resp *gcom.GetStackRegions200Response
	err := common.RetryRequest(ctx, "list stack regions", func() (*http.Response, error) {
		r, httpResp, err := client.StackRegionsAPI.GetStackRegions(ctx).Execute()
		resp = r
		return httpResp, err
	})
	return resp, err
}

// accessPolicyQuery narrows a list-access-policies request. A zero-valued field is omitted from
// the request, so callers only set the filters they need (Region is always required by the API).
type accessPolicyQuery struct {
	Region string
	OrgID  *int32
	Name   string
}

// listAccessPoliciesWithRetry lists access policies matching q, retrying transient grafana.com
// errors.
func listAccessPoliciesWithRetry(ctx context.Context, client *gcom.APIClient, q accessPolicyQuery) (*gcom.GetAccessPolicies200Response, error) {
	var resp *gcom.GetAccessPolicies200Response
	err := common.RetryRequest(ctx, "list access policies", func() (*http.Response, error) {
		req := client.AccesspoliciesAPI.GetAccessPolicies(ctx).Region(q.Region)
		if q.OrgID != nil {
			req = req.OrgId(*q.OrgID)
		}
		if q.Name != "" {
			req = req.Name(q.Name)
		}
		r, httpResp, err := req.Execute()
		resp = r
		return httpResp, err
	})
	return resp, err
}
