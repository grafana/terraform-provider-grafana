package fleetmanagement

import (
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/fleetmanagementapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Config type constants for pipeline and collector resources
const (
	ConfigTypeAlloy = "ALLOY"
	ConfigTypeOtel  = "OTEL"
)

func withClientForDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) (*fleetmanagementapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Data Source Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.FleetManagementClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Fleet Management API.",
			"Please ensure that fleet_management_auth and fleet_management_url are set in the provider configuration.",
		)

		return nil, fmt.Errorf("the Fleet Management client is nil")
	}

	return client.FleetManagementClient, nil
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*fleetmanagementapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil, fmt.Errorf("unexpected Resource Configure Type: %T, expected *common.Client", req.ProviderData)
	}

	if client.FleetManagementClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Fleet Management API.",
			"Please ensure that fleet_management_auth and fleet_management_url are set in the provider configuration.",
		)

		return nil, fmt.Errorf("the Fleet Management client is nil")
	}

	return client.FleetManagementClient, nil
}
