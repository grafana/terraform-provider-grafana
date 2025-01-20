package v0alpha1

import (
	"github.com/grafana/terraform-provider-grafana/appplatform/pkg/client"
)

// Client is a dashboard resource client.
type Client = *client.NamespacedClient[*Dashboard, *DashboardList]

// NewOrgClient is a dashboard resource client for the on-prem Grafana instance.
func NewOrgClient(reg client.Registry, orgID int64) (Client, error) {
	cli, err := reg.ClientFor(kindDashboard)
	if err != nil {
		return nil, err
	}

	return client.NewNamespaced(
		client.NewResourceClient[*Dashboard, *DashboardList](cli, Kind()),
		orgID, true,
	), nil
}

// NewCloudClient is a dashboard resource client for the Grafana Cloud stack instance.
func NewCloudClient(reg client.Registry, stackID int64) (Client, error) {
	cli, err := reg.ClientFor(kindDashboard)
	if err != nil {
		return nil, err
	}

	return client.NewNamespaced(
		client.NewResourceClient[*Dashboard, *DashboardList](cli, Kind()),
		stackID, false,
	), nil
}
