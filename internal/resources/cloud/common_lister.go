package cloud

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// ListerData is used as the data arg in "ListIDs" functions. It allows getting data common to multiple resources.
type ListerData struct {
	orgSlug string

	orgID   int32
	orgInit sync.Once

	stacks     []gcom.FormattedApiInstance
	stacksInit sync.Once
}

func NewListerData(orgSlug string) *ListerData {
	return &ListerData{orgSlug: orgSlug}
}

func (d *ListerData) Stacks(ctx context.Context, client *gcom.APIClient) ([]gcom.FormattedApiInstance, error) {
	var err error
	d.stacksInit.Do(func() {
		stacksReq := client.InstancesAPI.GetInstances(ctx)
		var stacksResp *gcom.GetInstances200Response
		err = RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
			var hr *http.Response
			stacksResp, hr, err = stacksReq.Execute()
			return hr, err
		})
		if err != nil {
			return
		}
		d.stacks = stacksResp.Items
	})
	return d.stacks, err
}

func (d *ListerData) OrgSlug() string {
	return d.orgSlug
}

func (d *ListerData) OrgID(ctx context.Context, client *gcom.APIClient) (int32, error) {
	var err error
	d.orgInit.Do(func() {
		org := d.OrgSlug()
		orgReq := client.OrgsAPI.GetOrg(ctx, org)
		var orgResp *gcom.FormattedApiOrgPublic
		err = RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
			var hr *http.Response
			orgResp, hr, err = orgReq.Execute()
			return hr, err
		})
		if err != nil {
			return
		}
		d.orgID = int32(orgResp.Id)
	})
	return d.orgID, err
}

// cloudListerFunction is a helper function that wraps a lister function be used more easily in cloud resources.
func cloudListerFunction(listerFunc func(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error)) common.ResourceListIDsFunc {
	return func(ctx context.Context, client *common.Client, data any) ([]string, error) {
		lm, ok := data.(*ListerData)
		if !ok {
			return nil, fmt.Errorf("unexpected data type: %T", data)
		}
		if client.GrafanaCloudAPI == nil {
			return nil, fmt.Errorf("client not configured for Grafana Cloud API")
		}
		return listerFunc(ctx, client.GrafanaCloudAPI, lm)
	}
}
