package v0alpha1

import (
	"github.com/grafana/terraform-provider-grafana/appplatform/pkg/client"
)

// Client is a dashboard resource client.
type Client = *client.NamespacedClient[*Dashboard, *DashboardList]

// NewClient is a dashboard resource client for the Grafana Cloud stack instance.
func NewClient(reg client.Registry, stackOrOrgID int64, isOrg bool) (Client, error) {
	cli, err := reg.ClientFor(kindDashboard)
	if err != nil {
		return nil, err
	}

	return client.NewNamespaced(
		client.NewResourceClient[*Dashboard, *DashboardList](cli, Kind()),
		stackOrOrgID, isOrg,
	), nil
}
