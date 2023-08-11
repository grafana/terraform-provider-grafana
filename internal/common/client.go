package common

import (
	"sync"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/machine-learning-go-client/mlapi"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"

	"github.com/grafana/terraform-provider-grafana/internal/common/connections"
)

type Client struct {
	GrafanaAPIURL    string
	GrafanaAPIConfig *gapi.Config
	GrafanaAPI       *gapi.Client
	GrafanaCloudAPI  *gapi.Client

	SMAPI *SMAPI.Client

	MLAPI *mlapi.Client

	OnCallClient *onCallAPI.Client

	ConnectionsAPI *connections.Client

	AlertingMutex sync.Mutex
}
