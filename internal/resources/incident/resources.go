package incident

import (
	"errors"
	"fmt"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var Resources = []*common.Resource{
	makeResourceRole().WithLister(listRoleIDs),
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*incident.Client, error) {
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil, fmt.Errorf("unexpected Resource Configure Type: %T", req.ProviderData)
	}

	if client.IncidentClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana Incident API.",
			"Please ensure that url and auth are set in the provider configuration and the Grafana IRM app is installed.",
		)
		return nil, errors.New("IncidentClient is nil")
	}

	return client.IncidentClient, nil
}
