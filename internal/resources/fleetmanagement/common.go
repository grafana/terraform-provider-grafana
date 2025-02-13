package fleetmanagement

import (
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/fleetmanagementapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

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
