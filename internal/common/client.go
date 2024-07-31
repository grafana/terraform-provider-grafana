package common

import (
	"context"
	"net/url"
	"strings"
	"sync"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	gcom "github.com/grafana/grafana-com-public-clients/go"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/machine-learning-go-client/mlapi"
	slo "github.com/grafana/slo-openapi-client/go"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Client struct {
	GrafanaAPIURL       string
	GrafanaAPIURLParsed *url.URL
	GrafanaAPI          *goapi.GrafanaHTTPAPI
	GrafanaAPIConfig    *goapi.TransportConfig

	GrafanaCloudAPI *gcom.APIClient
	SMAPI           *SMAPI.Client
	MLAPI           *mlapi.Client
	OnCallClient    *onCallAPI.Client
	SLOClient       *slo.APIClient

	alertingMutex sync.Mutex
}

// WithAlertingMutex is a helper function that wraps a CRUD Terraform function with a mutex.
func WithAlertingMutex[T schema.CreateContextFunc | schema.ReadContextFunc | schema.UpdateContextFunc | schema.DeleteContextFunc](f T) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		lock := &meta.(*Client).alertingMutex
		lock.Lock()
		defer lock.Unlock()
		return f(ctx, d, meta)
	}
}

func (c *Client) GrafanaSubpath(path string) string {
	path = strings.TrimPrefix(path, c.GrafanaAPIURLParsed.Path)
	return c.GrafanaAPIURLParsed.JoinPath(path).String()
}
