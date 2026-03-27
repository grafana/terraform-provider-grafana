package common

import (
	"context"
	"net/url"
	"strings"
	"sync"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/grafana-app-sdk/k8s"
	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/k6-cloud-openapi-client-go/k6"
	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/slo-openapi-client/go/slo"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudintegrationsapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudproviderapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/connectionsapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/fleetmanagementapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/k6providerapi"
)

type Client struct {
	GrafanaAPIURL                 string
	GrafanaAPIURLParsed           *url.URL
	GrafanaAPI                    *goapi.GrafanaHTTPAPI
	GrafanaAPIConfig              *goapi.TransportConfig
	GrafanaAppPlatformAPI         *k8s.ClientRegistry
	GrafanaAppPlatformAPIClientID string
	GrafanaOrgID                  int64
	GrafanaStackID                int64

	GrafanaCloudAPI       *gcom.APIClient
	SMAPI                 *SMAPI.Client
	MLAPI                 *mlapi.Client
	OnCallClient          *onCallAPI.Client
	SLOClient             *slo.APIClient
	CloudIntegrationsAPIClient *cloudintegrationsapi.Client
	CloudProviderAPI           *cloudproviderapi.Client
	ConnectionsAPIClient       *connectionsapi.Client
	FleetManagementClient *fleetmanagementapi.Client
	FrontendO11yAPIClient *frontendo11yapi.Client
	AssertsAPIClient      *assertsapi.APIClient

	K6APIClient *k6.APIClient
	K6APIConfig *k6providerapi.K6APIConfig

	alertingMutex  sync.Mutex
	folderMutex    sync.Mutex
	dashboardMutex sync.Mutex
}

// WithAlertingMutex is a helper function that wraps a CRUD Terraform function with a mutex.
func WithAlertingMutex[T schema.CreateContextFunc | schema.ReadContextFunc | schema.UpdateContextFunc | schema.DeleteContextFunc](f T) T {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		lock := &meta.(*Client).alertingMutex
		lock.Lock()
		defer lock.Unlock()
		return f(ctx, d, meta)
	}
}

// WithAlertingLock runs f while holding the alerting mutex. Used by Plugin Framework resources that need to serialize alerting API calls.
func (c *Client) WithAlertingLock(f func()) {
	c.alertingMutex.Lock()
	defer c.alertingMutex.Unlock()
	f()
}

// WithFolderMutex is a helper function that wraps a CRUD Terraform function with a mutex.
func WithFolderMutex[T schema.CreateContextFunc | schema.ReadContextFunc | schema.UpdateContextFunc | schema.DeleteContextFunc](f T) T {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		lock := &meta.(*Client).folderMutex
		lock.Lock()
		defer lock.Unlock()
		return f(ctx, d, meta)
	}
}

// WithFolderLock runs f while holding the folder mutex. Used by Plugin Framework resources that need to serialize folder API calls.
func (c *Client) WithFolderLock(f func()) {
	c.folderMutex.Lock()
	defer c.folderMutex.Unlock()
	f()
}

// WithDashboardMutex is a helper function that wraps a CRUD Terraform function with a mutex.
func WithDashboardMutex[T schema.CreateContextFunc | schema.ReadContextFunc | schema.UpdateContextFunc | schema.DeleteContextFunc](f T) T {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		lock := &meta.(*Client).dashboardMutex
		lock.Lock()
		defer lock.Unlock()
		return f(ctx, d, meta)
	}
}

// WithDashboardLock runs f while holding the dashboard mutex. Used by Plugin Framework resources that need to serialize dashboard API calls.
func (c *Client) WithDashboardLock(f func()) {
	c.dashboardMutex.Lock()
	defer c.dashboardMutex.Unlock()
	f()
}

func (c *Client) GrafanaSubpath(path string) string {
	path = strings.TrimPrefix(path, c.GrafanaAPIURLParsed.Path)
	return c.GrafanaAPIURLParsed.JoinPath(path).String()
}
