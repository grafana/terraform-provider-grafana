package v2alpha1

import (
	"github.com/grafana/terraform-provider-grafana/appplatform/pkg/client"
)

// NewOrgClient is a dashboard resource client for the on-prem Grafana instance.
func NewOrgClient(reg client.Registry, orgID int64) (*client.NamespacedClient[*Dashboard, *DashboardList], error) {
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
func NewCloudClient(reg client.Registry, stackID int64) (*client.NamespacedClient[*Dashboard, *DashboardList], error) {
	cli, err := reg.ClientFor(kindDashboard)
	if err != nil {
		return nil, err
	}

	return client.NewNamespaced(
		client.NewResourceClient[*Dashboard, *DashboardList](cli, Kind()),
		stackID, false,
	), nil
}
