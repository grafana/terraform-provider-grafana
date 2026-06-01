package assistant

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

var Resources = []*common.Resource{
	makeResourceRule(),
	makeResourceSkill(),
	makeResourceQuickstart(),
	makeResourceMCPServer(),
}

func withClientForResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*assistantapi.Client, error) {
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil, fmt.Errorf("unexpected Resource Configure Type: %T", req.ProviderData)
	}

	if client.AssistantAPIClient == nil {
		resp.Diagnostics.AddError(
			"The Grafana Provider is missing a configuration for the Grafana Assistant API.",
			"Please ensure that url and auth are set in the provider configuration and the Grafana Assistant app is installed.",
		)
		return nil, errors.New("AssistantAPIClient is nil")
	}

	return client.AssistantAPIClient, nil
}
