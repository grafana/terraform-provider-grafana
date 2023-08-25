package common

import (
	"sync"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	gapi "github.com/grafana/grafana-api-golang-client"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/machine-learning-go-client/mlapi"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"
)

type Client struct {
	GrafanaAPIURL    string
	GrafanaAPIConfig *gapi.Config
	GrafanaAPI       *gapi.Client
	GrafanaCloudAPI  *gapi.Client

	GrafanaOAPI *goapi.GrafanaHTTPAPI

	SMAPI *SMAPI.Client

	MLAPI *mlapi.Client

	OnCallClient *onCallAPI.Client

	AlertingMutex sync.Mutex
}
