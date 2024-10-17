package connections

import (
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/connectionsapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var DataSources = []*common.DataSource{
	makeDatasourceMetricsEndpointScrapeJob(),
}

var Resources = []*common.Resource{
	makeResourceMetricsEndpointScrapeJob(),
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*connectionsapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.ConnectionsAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Connections API.",
			"Please ensure that connections_url and connections_access_token are set in the provider configuration.",
		)

		return nil, fmt.Errorf("ConnectionsAPI is nil")
	}

	return client.ConnectionsAPIClient, nil
}

func withClientForDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) (*connectionsapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected DataSource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.ConnectionsAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Cloud Provider API.",
			"Please ensure that connections_api_url and connections_access_token are set in the provider configuration.",
		)

		return nil, fmt.Errorf("ConnectionsAPI is nil")
	}

	return client.ConnectionsAPIClient, nil
}
