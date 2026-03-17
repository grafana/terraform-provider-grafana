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
	CloudProviderAPI      *cloudproviderapi.Client
	ConnectionsAPIClient  *connectionsapi.Client
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

// WithDashboardMutex is a helper function that wraps a CRUD Terraform function with a mutex.
func WithDashboardMutex[T schema.CreateContextFunc | schema.ReadContextFunc | schema.UpdateContextFunc | schema.DeleteContextFunc](f T) T {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		lock := &meta.(*Client).dashboardMutex
		lock.Lock()
		defer lock.Unlock()
		return f(ctx, d, meta)
	}
}

func (c *Client) GrafanaSubpath(path string) string {
	path = strings.TrimPrefix(path, c.GrafanaAPIURLParsed.Path)
	return c.GrafanaAPIURLParsed.JoinPath(path).String()
}

// frameworkProviderClient is set by the Plugin Framework provider's Configure and
// used as a fallback when a Framework resource's Configure was not called with
// ProviderData (e.g. due to mux/ordering). Access via Get/SetFrameworkProviderClient.
var (
	frameworkProviderClient   *Client
	frameworkProviderClientMu sync.RWMutex
)

// SetFrameworkProviderClient stores the client from the Framework provider's Configure.
// It is used by Framework resources when their Configure was not called with ProviderData.
func SetFrameworkProviderClient(c *Client) {
	frameworkProviderClientMu.Lock()
	defer frameworkProviderClientMu.Unlock()
	frameworkProviderClient = c
}

// GetFrameworkProviderClient returns the client set by SetFrameworkProviderClient, or nil.
func GetFrameworkProviderClient() *Client {
	frameworkProviderClientMu.RLock()
	defer frameworkProviderClientMu.RUnlock()
	return frameworkProviderClient
}

// EnsureFrameworkProviderClientFromEnvFunc is set by the provider so that Framework resources
// can trigger "create client from env and set fallback" when the fallback is nil (e.g. CI mux/ordering).
var (
	ensureFrameworkProviderClientFromEnvFunc   func() error
	ensureFrameworkProviderClientFromEnvFuncMu sync.RWMutex
)

// RegisterEnsureFrameworkProviderClientFromEnv registers the provider's "create client from env" function.
// Called from pkg/provider so that internal/resources can trigger it without importing the provider.
func RegisterEnsureFrameworkProviderClientFromEnv(f func() error) {
	ensureFrameworkProviderClientFromEnvFuncMu.Lock()
	defer ensureFrameworkProviderClientFromEnvFuncMu.Unlock()
	ensureFrameworkProviderClientFromEnvFunc = f
}

// EnsureFrameworkProviderClientFromEnv calls the registered function to create a client from env
// and set the Framework fallback. Used by Framework resources (e.g. grafana_user) when the fallback is nil.
func EnsureFrameworkProviderClientFromEnv() error {
	ensureFrameworkProviderClientFromEnvFuncMu.RLock()
	f := ensureFrameworkProviderClientFromEnvFunc
	ensureFrameworkProviderClientFromEnvFuncMu.RUnlock()
	if f == nil {
		return nil
	}
	return f()
}
